package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/puppetlabs/wash/api"
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
	go api.StartAPI(registry, "wash-api.sock")

	if err := fuse.ServeFuseFS(registry, mountpoint, *debug); err != nil {
		log.Warnf("%v", err)
		os.Exit(1)
	}
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
