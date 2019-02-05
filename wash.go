package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/gcp"
	"github.com/puppetlabs/wash/kubernetes"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

var progName = filepath.Base(os.Args[0])
var debug = flag.Bool("debug", false, "Enable debug output from FUSE")
var slow = flag.Bool("slow", false, "Disable prefetch on files and directories to reduce network activity")

func usage() {
	fmt.Fprintf(os.Stderr, "%s mounts remote resources with FUSE", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.Init(*debug)

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	if err := mount(mountpoint); err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}
}

type pluginInit struct {
	name   string
	plugin plugin.DirProtocol
	err    error
}

type instantiator = func(string, interface{}, *bigcache.BigCache) (plugin.DirProtocol, error)
type instData struct {
	instantiator
	context interface{}
}

func mount(mountpoint string) error {
	config := bigcache.DefaultConfig(plugin.DefaultTimeout)
	config.CleanWindow = 1 * time.Second
	cache, err := bigcache.NewBigCache(config)
	if err != nil {
		return err
	}

	if *debug {
		fuse.Debug = func(msg interface{}) {
			log.Debugf("%v", msg)
		}
	}
	plugin.Init(*slow)

	pluginInstantiators := map[string]instData{
		"docker": {docker.Create, nil},
		"gcp":    {gcp.Create, nil},
	}

	k8sContexts, err := kubernetes.ListContexts()
	if err != nil {
		return err
	}
	for name, context := range k8sContexts {
		pluginInstantiators["kubernetes_"+name] = instData{kubernetes.Create, context}
	}

	plugins := make(chan pluginInit)
	for k, v := range pluginInstantiators {
		go func(name string, create instData) {
			log.Printf("Loading %v integration", name)
			pluginInst, err := create.instantiator(name, create.context, cache)
			plugins <- pluginInit{name, pluginInst, err}
		}(k, v)
	}

	log.Printf("Mounting at %v", mountpoint)
	fuseServer, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer fuseServer.Close()

	pluginMap := make(map[string]plugin.DirProtocol)
	for range pluginInstantiators {
		pluginInst := <-plugins
		if pluginInst.err != nil {
			log.Printf("Error loading %v: %v", pluginInst.name, pluginInst.err)
		} else {
			log.Printf("Loaded %v", pluginInst.name)
			pluginMap[pluginInst.name] = pluginInst.plugin
		}
	}

	if len(pluginMap) == 0 {
		return errors.New("No plugins loaded")
	}

	log.Printf("Serving filesystem")
	filesys := &plugin.FS{Plugins: pluginMap}
	if err := fs.Serve(fuseServer, filesys); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-fuseServer.Ready
	if err := fuseServer.MountError; err != nil {
		return err
	}
	log.Printf("Done")

	return nil
}
