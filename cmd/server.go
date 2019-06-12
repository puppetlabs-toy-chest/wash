package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/cmd/internal/config"
	"github.com/puppetlabs/wash/cmd/internal/server"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/plugin/aws"
	"github.com/puppetlabs/wash/plugin/docker"
	"github.com/puppetlabs/wash/plugin/kubernetes"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var internalPlugins = map[string]plugin.Root{
	"aws":        &aws.Root{},
	"docker":     &docker.Root{},
	"kubernetes": &kubernetes.Root{},
}

func serverCommand() *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server <mountpoint>",
		Short: "Sets up the Wash daemon (API and FUSE servers)",
		Long: `Initializes all of the plugins, then sets up the Wash daemon (its API and FUSE servers).
To stop it, make sure you're not using the filesystem at <mountpoint>, then enter Ctrl-C.`,
		Args:   cobra.MinimumNArgs(1),
		PreRun: bindServerArgs,
		RunE:   toRunE(serverMain),
	}
	addServerArgs(serverCmd, "info")

	return serverCmd
}

func serverMain(cmd *cobra.Command, args []string) exitCode {
	mountpoint := args[0]
	mountpoint, err := filepath.Abs(mountpoint)
	if err != nil {
		cmdutil.ErrPrintf("Could not compute the absolute path of the mountpoint %v: %v", mountpoint, err)
		return exitCode{1}
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})

	// On Ctrl-C, trigger the clean-up. This consists of shutting down the API
	// server and unmounting the FS.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	plugins, serverOpts, err := serverOptsFor(cmd)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	srv := server.New(mountpoint, config.Socket, plugins, serverOpts)
	if err := srv.Start(); err != nil {
		log.Warn(err)
		return exitCode{1}
	}
	srv.Wait(sigCh)
	return exitCode{0}
}

func addServerArgs(cmd *cobra.Command, defaultLogLevel string) {
	cmd.Flags().String("loglevel", defaultLogLevel, "Set the logging level")
	cmd.Flags().String("logfile", "", "Set the log file's location. Defaults to stdout")
	cmd.Flags().String("cpuprofile", "", "Write cpu profile to file")
	cmd.Flags().String("config-file", config.DefaultFile(), "Set the config file's location")
}

func bindServerArgs(cmd *cobra.Command, args []string) {
	// Only bind config lookup when invoking the specific command as viper bindings are global.
	errz.Fatal(viper.BindPFlag("loglevel", cmd.Flags().Lookup("loglevel")))
	errz.Fatal(viper.BindPFlag("logfile", cmd.Flags().Lookup("logfile")))
	errz.Fatal(viper.BindPFlag("cpuprofile", cmd.Flags().Lookup("cpuprofile")))
}

// serverOptsFor returns map of plugins and server.Opts for the given command.
func serverOptsFor(cmd *cobra.Command) (map[string]plugin.Root, server.Opts, error) {
	// Read the config
	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		panic(err.Error())
	}
	if err := config.ReadFrom(configFile); err != nil {
		return nil, server.Opts{}, err
	}

	// Unmarshal the external plugins, if any are specified
	var externalPlugins []plugin.ExternalPluginSpec
	if err := viper.UnmarshalKey("external-plugins", &externalPlugins); err != nil {
		return nil, server.Opts{}, fmt.Errorf("failed to unmarshal the external-plugins key: %v", err)
	}

	// Load internal plugins that are not specifically excluded.
	enabledPlugins := viper.GetStringSlice("plugins")
	plugins := make(map[string]plugin.Root)
	if len(enabledPlugins) > 0 {
		for _, name := range enabledPlugins {
			if plug, ok := internalPlugins[name]; ok {
				plugins[name] = plug
			} else {
				log.Warnf("Requested unknown plugin %s", name)
			}
		}
	} else {
		// Copy internalPlugins so we don't mutate it.
		for name, plug := range internalPlugins {
			plugins[name] = plug
		}
	}

	// Ensure external plugins are valid scripts and convert them to plugin.Root types.
	for _, spec := range externalPlugins {
		intPlugin, err := spec.Load()
		if err != nil {
			log.Warnf("%v failed to load: %+v", spec.Script, err)
			continue
		}

		name := plugin.Name(intPlugin)
		if _, ok := plugins[name]; ok {
			log.Warnf("Overriding plugin %s with external plugin %s", name, spec.Script)
		}
		plugins[name] = intPlugin
	}

	config := make(map[string]map[string]interface{})
	for name := range plugins {
		config[name] = viper.GetStringMap(name)
	}

	// Return the options
	return plugins, server.Opts{
		CPUProfilePath: viper.GetString("cpuprofile"),
		LogFile:        viper.GetString("logfile"),
		LogLevel:       viper.GetString("loglevel"),
		PluginConfig:   config,
	}, nil
}
