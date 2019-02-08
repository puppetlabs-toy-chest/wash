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

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/kubernetes"
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
	errz.Fatal(viper.BindPFlag("loglevel", serverCmd.Flags().Lookup("loglevel")))

	serverCmd.Flags().String("logfile", "", "Set the log file's location. Defaults to stdout")
	errz.Fatal(viper.BindPFlag("logfile", serverCmd.Flags().Lookup("logfile")))

	serverCmd.RunE = toRunE(serverMain)

	return serverCmd
}

func serverMain(cmd *cobra.Command, args []string) exitCode {
	mountpoint := args[0]
	loglevel := viper.GetString("loglevel")
	logfile := viper.GetString("logfile")

	logFH, err := initializeLogger(loglevel, logfile)
	if err != nil {
		fmt.Printf("Failed to initialize the logger: %v\n", err)
		return exitCode{1}
	}
	if logFH != nil {
		defer func() { plugin.LogErr(logFH.Close()) }()
	}

	registry, err := initializePlugins()
	if err != nil {
		log.Warn(err)
		return exitCode{1}
	}

	plugin.InitCache()

	apiServerStopCh, apiServerStoppedCh, err := api.StartAPI(registry, config.Socket)
	if err != nil {
		log.Warn(err)
		return exitCode{1}
	}
	stopAPIServer := func() {
		// Shutdown the API server; wait for the shutdown to finish
		apiShutdownDeadline := time.Now().Add(3 * time.Second)
		apiShutdownCtx, cancelFunc := context.WithDeadline(context.Background(), apiShutdownDeadline)
		defer cancelFunc()
		apiServerStopCh <- apiShutdownCtx
		<-apiServerStoppedCh
	}

	fuseServerStopCh, fuseServerStoppedCh, err := fuse.ServeFuseFS(registry, mountpoint)
	if err != nil {
		stopAPIServer()
		log.Warn(err)
		return exitCode{1}
	}
	stopFUSEServer := func() {
		// Shutdown the FUSE server; wait for the shutdown to finish
		fuseServerStopCh <- true
		<-fuseServerStoppedCh
	}

	// On Ctrl-C, trigger the clean-up. This consists of shutting down the API
	// server and unmounting the FS.
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT)

	select {
	case <-sigCh:
		stopAPIServer()
		stopFUSEServer()
	case <-fuseServerStoppedCh:
		// This code-path is possible if the FUSE server prematurely shuts down, which
		// can happen if the user unmounts the mountpoint while the server's running.
		stopAPIServer()
	case <-apiServerStoppedCh:
		// This code-path is possible if the API server prematurely shuts down
		stopFUSEServer()
	}

	return exitCode{0}
}

var levelMap = map[string]log.Level{
	"warn":  log.WarnLevel,
	"info":  log.InfoLevel,
	"debug": log.DebugLevel,
	"trace": log.TraceLevel,
}

func initializeLogger(levelStr string, logfile string) (*os.File, error) {
	level, ok := levelMap[levelStr]
	if !ok {
		var allLevels []string
		for level := range levelMap {
			allLevels = append(allLevels, level)
		}

		err := fmt.Errorf(
			"%v is not a valid level. Valid levels are %v",
			level,
			strings.Join(allLevels, ", "),
		)

		return nil, err
	}

	log.SetLevel(level)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})

	var logFH *os.File
	if logfile != "" {
		logFH, err := os.Create(logfile)
		if err != nil {
			return nil, err
		}

		log.SetOutput(logFH)
	}

	return logFH, nil
}

type pluginInit struct {
	name   string
	plugin plugin.Entry
	err    error
}

func initializePlugins() (*plugin.Registry, error) {
	plugins := make(map[string]plugin.Root)
	for _, plugin := range []plugin.Root{
		&docker.Root{},
		&kubernetes.Root{},
	} {
		if err := plugin.Init(); err != nil {
			return nil, err
		}
		plugins[plugin.Name()] = plugin
	}

	if len(plugins) == 0 {
		return nil, errors.New("No plugins loaded")
	}

	return &plugin.Registry{Plugins: plugins}, nil
}
