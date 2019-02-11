package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/fuse"
)

var progName = filepath.Base(os.Args[0])
var debug = flag.Bool("debug", false, "Enable debug output")
var quiet = flag.Bool("quiet", false, "Suppress operational logging and only log errors")
var slow = flag.Bool("slow", false, "Disable prefetch on files and directories to reduce network activity")

func usage() {
	fmt.Fprintf(os.Stderr, "%s mounts remote resources with FUSE\n", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.Init(*debug, *quiet)

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	registry, err := initializePlugins()
	if err != nil {
		log.Warnf("%v", err)
		os.Exit(1)
	}

	mountpoint := flag.Arg(0)
	go startAPI(registry, "wash-api.sock")

	if err := fuse.ServeFuseFS(registry, mountpoint, *debug); err != nil {
		log.Warnf("%v", err)
		os.Exit(1)
	}
}

func startAPI(registry *plugin.Registry, socketPath string) error {
	log.Printf("API: started")

	if _, err := os.Stat(socketPath); err == nil {
		// Socket already exists, so nuke it and recreate it
		log.Printf("API: Cleaning up old socket")
		if err := os.Remove(socketPath); err != nil {
			log.Warnf("API: %v", err)
			return err
		}
	}

	server, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Warnf("API: %v", err)
		return err
	}

	for {
		conn, err := server.Accept()
		log.Printf("API: accepted connection")
		if err != nil {
			log.Warnf("API: %v", err)
			return err
		}
		go func() {
			if err := handleAPIRequest(conn, registry); err != nil {
				log.Warnf("API: %v", err)
			}
		}()
	}
}

func handleAPIRequest(conn net.Conn, registry *plugin.Registry) error {
	defer conn.Close()

	// TODO: Fill in with an actual API

	return nil
}

type pluginInit struct {
	name   string
	plugin plugin.Entry
	err    error
}

type instantiator = func(string, interface{}, *datastore.MemCache) (plugin.Entry, error)
type instData struct {
	instantiator
	context interface{}
}

func initializePlugins() (*plugin.Registry, error) {
	cache, err := datastore.NewMemCache()
	if err != nil {
		return nil, err
	}

	pluginInstantiators := map[string]instData{
		"docker": {docker.Create, nil},
	}

	plugins := make(chan pluginInit)
	for k, v := range pluginInstantiators {
		go func(name string, create instData) {
			log.Printf("Loading %v integration", name)
			pluginInst, err := create.instantiator(name, create.context, cache)
			plugins <- pluginInit{name, pluginInst, err}
		}(k, v)
	}

	pluginMap := make(map[string]plugin.Entry)
	for range pluginInstantiators {
		pluginInst := <-plugins
		if pluginInst.err != nil {
			log.Warnf("Error loading %v: %v", pluginInst.name, pluginInst.err)
		} else {
			log.Warnf("Loaded %v", pluginInst.name)
			pluginMap[pluginInst.name] = pluginInst.plugin
		}
	}

	if len(pluginMap) == 0 {
		return nil, errors.New("No plugins loaded")
	}

	return &plugin.Registry{Plugins: pluginMap}, nil
}
