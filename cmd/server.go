package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/plugin"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func serverCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server <mountpoint>",
		Short: "Sets up the Wash API and FUSE servers",
		Long:  "Initializes all of the plugins, then sets up the Wash API and FUSE servers",
		Args:  cobra.MinimumNArgs(1),
	}

	serverCmd.Flags().String("loglevel", "info", "Set the logging level")
	viper.BindPFlag("loglevel", serverCmd.Flags().Lookup("loglevel"))

	serverCmd.RunE = toRunE(serverMain)

	return serverCmd
}

func serverMain(cmd *cobra.Command, args []string) exitCode {
	mountpoint := args[0]
	loglevel := viper.GetString("loglevel")

	initializeLogger(loglevel)

	registry, err := initializePlugins()
	if err != nil {
		log.Warnf("%v", err)
		return exitCode{1}
	}

	apiServerStopCh, err := api.StartAPI(registry, config.Socket)
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

	fuseServerStopCh, err := fuse.ServeFuseFS(registry, mountpoint)
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

var levelMap = map[string]log.Level{
	"warn":  log.WarnLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"trace": log.TraceLevel,
}

func initializeLogger(levelStr string) {
	level, ok := levelMap[levelStr]
	if !ok {
		var allLevels []string
		for level := range levelMap {
			allLevels = append(allLevels, level)
		}

		panic(fmt.Sprintf(
			"%v is not a valid level. Valid levels are %v",
			level,
			strings.Join(allLevels, ", ")),
		)
	}

	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
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
