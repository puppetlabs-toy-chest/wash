package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/fuse"
)

var progName = filepath.Base(os.Args[0])
var debug = flag.Bool("debug", false, "Enable debug output")
var quiet = flag.Bool("quiet", false, "Suppress operational logging and only log errors")

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

func initializePlugins() (*plugin.Registry, error) {
	plugins := make(map[string]plugin.Root)

	plugins["docker"] = &docker.Root{}

	for _, plugin := range plugins {
		if err := plugin.Init(); err != nil {
			return nil, err
		}
	}

	if len(plugins) == 0 {
		return nil, errors.New("No plugins loaded")
	}

	return &plugin.Registry{Plugins: plugins}, nil
}
