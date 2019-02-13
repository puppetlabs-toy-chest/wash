package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"
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

	apiServerStopCh, err := api.StartAPI(registry, "/tmp/wash-api.sock")
	if err != nil {
		log.Warnf("%v", err)
		os.Exit(1)
	}
	stopAPIServer := func() {
		// Shutdown the API server; wait for the shutdown to finish
		apiShutdownDeadline := time.Now().Add(3 * time.Second)
		apiShutdownCtx, cancelFunc := context.WithDeadline(context.Background(), apiShutdownDeadline)
		defer cancelFunc()
		apiServerStopCh <- apiShutdownCtx
		<-apiServerStopCh
	}

	mountpoint := flag.Arg(0)
	fuseServerStopCh, err := fuse.ServeFuseFS(registry, mountpoint, *debug)
	if err != nil {
		log.Warnf("%v", err)
		stopAPIServer()
		os.Exit(1)
	}

	// On Ctrl-C, trigger the clean-up. This consists of shutting down the API
	// server and unmounting the FS.
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT)

	<-sigCh

	stopAPIServer()

	// Shutdown the FUSE server; wait for the shutdown to finish
	fuseServerStopCh <- true
	<-fuseServerStopCh
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
