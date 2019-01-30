package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/gcp"
	"github.com/puppetlabs/wash/kubernetes"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
)

var progName = filepath.Base(os.Args[0])
var debug = flag.Bool("debug", false, "Enable debug output from FUSE")

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

type clientInit struct {
	name   string
	client plugin.DirProtocol
	err    error
}

type instantiator = func(string) (plugin.DirProtocol, error)

func mount(mountpoint string) error {
	clients := make(chan clientInit)

	if *debug {
		fuse.Debug = func(msg interface{}) {
			log.Debugf("%v", msg)
		}
	}

	clientInstantiators := map[string]instantiator{
		"docker":     docker.Create,
		"gcp":        gcp.Create,
		"kubernetes": kubernetes.Create,
	}

	for k, v := range clientInstantiators {
		go func(name string, create instantiator) {
			log.Printf("Loading %v integration", name)
			client, err := create(name)
			clients <- clientInit{name, client, err}
		}(k, v)
	}

	log.Printf("Mounting at %v", mountpoint)
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	clientMap := make(map[string]plugin.DirProtocol)
	for range clientInstantiators {
		client := <-clients
		if client.err != nil {
			log.Printf("Error loading %v: %v", client.name, client.err)
		} else {
			log.Printf("Loaded %v", client.name)
			clientMap[client.name] = client.client
		}
	}

	if len(clientMap) == 0 {
		return errors.New("No plugins loaded")
	}

	log.Printf("Serving filesystem")
	filesys := &plugin.FS{Clients: clientMap}
	if err := fs.Serve(c, filesys); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}
	log.Printf("Done")

	return nil
}
