package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/log"
	"github.com/puppetlabs/wash/plugin"

	"github.com/spf13/cobra"
)

func serverCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server <mountpoint>",
		Short: "Sets up the Wash API and FUSE servers",
		Long:  "Initializes all of the plugins, then sets up the Wash API and FUSE servers",
		Args:  cobra.MinimumNArgs(1),
	}

	serverCmd.Flags().Bool("debug", false, "Enable debug output")
	serverCmd.Flags().Bool("quiet", false, "Suppress operational logging and only log errors")

	serverCmd.RunE = toRunE(serverMain)

	return serverCmd
}

func serverMain(cmd *cobra.Command, args []string) exitCode {
	mountpoint := args[0]
	socket := config.Fields.Socket
	debug, _ := cmd.Flags().GetBool("debug")
	quiet, _ := cmd.Flags().GetBool("quiet")

	log.Init(debug, quiet)

	registry, err := initializePlugins()
	if err != nil {
		log.Warnf("%v", err)
		return exitCode{1}
	}

	apiServerStopCh, err := api.StartAPI(registry, socket)
	if err != nil {
		log.Warnf("%v", err)
		return exitCode{1}
	}
	stopAPIServer := func() {
		// Shutdown the API server; wait for the shutdown to finish
		apiShutdownDeadline := time.Now().Add(3 * time.Second)
		apiShutdownCtx, cancelFunc := context.WithDeadline(context.Background(), apiShutdownDeadline)
		defer cancelFunc()
		apiServerStopCh <- apiShutdownCtx
		<-apiServerStopCh
	}

	fuseServerStopCh, err := fuse.ServeFuseFS(registry, mountpoint, debug)
	if err != nil {
		stopAPIServer()
		log.Warnf("%v", err)
		return exitCode{1}
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

	return exitCode{0}
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
