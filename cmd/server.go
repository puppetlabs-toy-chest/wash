package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/api"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/fuse"
	"github.com/puppetlabs/wash/journal"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/aws"
	"github.com/puppetlabs/wash/plugin/docker"
	"github.com/puppetlabs/wash/plugin/kubernetes"

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

	serverCmd.Flags().String("external-plugins", "", "Specify the file to load any external plugins")
	errz.Fatal(viper.BindPFlag("external-plugins", serverCmd.Flags().Lookup("external-plugins")))

	serverCmd.Flags().String("cpuprofile", "", "Write cpu profile to file")
	errz.Fatal(viper.BindPFlag("cpuprofile", serverCmd.Flags().Lookup("cpuprofile")))

	serverCmd.RunE = toRunE(serverMain)

	return serverCmd
}

func serverMain(cmd *cobra.Command, args []string) exitCode {
	mountpoint := args[0]
	loglevel := viper.GetString("loglevel")
	logfile := viper.GetString("logfile")
	externalPluginsPath := viper.GetString("external-plugins")

	logFH, err := loadLogger(loglevel, logfile)
	if err != nil {
		cmdutil.ErrPrintf("Failed to load the logger: %v\n", err)
		return exitCode{1}
	}
	if logFH != nil {
		defer func() {
			if err := logFH.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing logger: %+v", err)
			}
		}()
	}

	// Close any open journals on shutdown to ensure remaining entries are flushed to disk.
	defer func() {
		journal.CloseAll()
	}()

	registry := plugin.NewRegistry()
	loadInternalPlugins(registry)
	if externalPluginsPath != "" {
		loadExternalPlugins(registry, externalPluginsPath)
	}
	if len(registry.Plugins()) == 0 {
		log.Warn("No plugins loaded")
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
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	cpuprofile := viper.GetString("cpuprofile")
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		errz.Fatal(pprof.StartCPUProfile(f))
		defer pprof.StopCPUProfile()
	}

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

func loadLogger(levelStr string, logfile string) (*os.File, error) {
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
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
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

func loadPlugin(registry *plugin.Registry, name string, root plugin.Root) {
	log.Infof("Loading %v", name)
	if err := registry.RegisterPlugin(root); err != nil {
		// %+v is a convention used by some errors to print additional context such as a stack trace
		log.Warnf("%v failed to load: %+v", name, err)
	}
}

func loadInternalPlugins(registry *plugin.Registry) {
	log.Info("Loading internal plugins")
	loadPlugin(registry, "aws", &aws.Root{})
	loadPlugin(registry, "docker", &docker.Root{})
	loadPlugin(registry, "kubernetes", &kubernetes.Root{})
	log.Info("Finished loading internal plugins")
}

func loadExternalPlugins(registry *plugin.Registry, externalPluginsPath string) {
	logError := func(err error) {
		log.Warnf("Failed to load external plugins: %v\n", err)
	}

	log.Infof("Loading external plugins")

	externalPluginsFH, err := os.Open(externalPluginsPath)
	if err != nil {
		logError(err)
		return
	}
	defer func() {
		if err := externalPluginsFH.Close(); err != nil {
			log.Infof("Error closing %v: %+v", externalPluginsPath, err)
		}
	}()

	d := yaml.NewDecoder(externalPluginsFH)
	var externalPlugins []plugin.ExternalPluginSpec
	if err := d.Decode(&externalPlugins); err != nil {
		logError(err)
		return
	}

	for _, p := range externalPlugins {
		log.Infof("Loading %v", p.Script)
		if err := registry.RegisterExternalPlugin(p); err != nil {
			// %+v is a convention used by some errors to print additional context such as a stack trace
			log.Warnf("%v failed to load: %+v", p.Script, err)
		}
	}

	log.Infof("Finished loading external plugins")
}
