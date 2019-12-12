package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/cmd/internal/config"
	"github.com/puppetlabs/wash/cmd/internal/server"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

	plugins := make(map[string]plugin.Root)

	// Check the internal plugins
	if viper.IsSet("plugins") || viper.IsSet("external-plugins") {
		enabledPlugins := viper.GetStringSlice("plugins")
		for _, name := range enabledPlugins {
			if plug, ok := server.InternalPlugins[name]; ok {
				plugins[name] = plug
			} else {
				log.Warnf("Requested unknown plugin %s", name)
			}
		}
	} else if !plugin.IsInteractive() {
		// This is an edge-case for a user but a common case for
		// CI. Thus, load all the plugins so that we don't break
		// the latter. Note that we copy server.InternalPlugins
		// so that we don't mutate it.
		log.Warnf("Running non-interactively without having set the 'plugins'/'external-plugins' keys in %v. Loading all core plugins by default", configFile)
		for name, plug := range server.InternalPlugins {
			plugins[name] = plug
		}
	} else {
		// Assume first-time user. First, we prompt them to get a list
		// of plugins that they wish to enable. Next, we write the
		// enabled plugins back to their specified config file.
		enabledPlugins := []string{}

		// Prompt them for the list of enabled plugins. This should look something
		// like
		//     Do you use docker? y
		//     aws? y
		//     kubernetes? y
		//     ...
		//
		firstPlugin := true
		for name, plug := range server.InternalPlugins {
			var prompt string
			if firstPlugin {
				firstPlugin = false
				prompt = fmt.Sprintf("Do you use %v (y/n)?", name)
			} else {
				prompt = fmt.Sprintf("%v?", name)
			}
			input, err := cmdutil.Prompt(prompt, cmdutil.YesOrNoP)
			if err != nil {
				return nil, server.Opts{}, err
			}
			if enabled := input.(bool); enabled {
				enabledPlugins = append(enabledPlugins, name)
				plugins[name] = plug
			}
		}
		cmdutil.Printf("The %v core plugins have been enabled.\n", strings.Join(enabledPlugins, ", "))

		// Now write the enabled plugins back to the specified config file. Note that we
		// do a raw append of "plugins: <enabled_plugins>" to preserve any existing data,
		// including comments. The append should be OK because we know that the config
		// file doesn't have a "plugins" key, so adding it will not mess anything up.
		//
		// Note that making this a function makes the code a bit easier to read. Inlining
		// it results in some nested if/else statements.
		writeEnabledPlugins := func() error {
			configFileAbsPath := configFile
			if configFileAbsPath == config.DefaultFile() {
				configFileAbsPath = config.DefaultFileAbsPath()
			}
			viper.Set("plugins", enabledPlugins)
			marshalledEnabledPlugins, err := yaml.Marshal(map[string][]string{"plugins": enabledPlugins})
			if err != nil {
				return err
			}
			f, err := os.OpenFile(configFileAbsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(marshalledEnabledPlugins)
			return err
		}
		if err := writeEnabledPlugins(); err != nil {
			log.Warnf("Failed to write-back the list of enabled plugins to %v: %v\n\n", configFile, err)
		} else {
			// The write was successful
			cmdutil.Printf("You can disable them by modifying the 'plugins' key in your config\nfile (%v), and then restarting the shell\n\n", configFile)
		}
	}

	// Check the external plugins. First unmarshal their spec, ensure that
	// they're valid scripts, then convert them to plugin.Root types.
	var externalPlugins []plugin.ExternalPluginSpec
	if err := viper.UnmarshalKey("external-plugins", &externalPlugins); err != nil {
		return nil, server.Opts{}, fmt.Errorf("failed to unmarshal the external-plugins key: %v", err)
	}
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

	pluginConfig := make(map[string]map[string]interface{})
	for name := range plugins {
		pluginConfig[name] = viper.GetStringMap(name)
	}

	// Return the options
	return plugins, server.Opts{
		CPUProfilePath: viper.GetString("cpuprofile"),
		LogFile:        viper.GetString("logfile"),
		LogLevel:       viper.GetString("loglevel"),
		PluginConfig:   pluginConfig,
	}, nil
}
