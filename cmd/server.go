package cmd

import (
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Benchkram/errz"
	"github.com/puppetlabs/wash/cmd/internal/server"
	cmdutil "github.com/puppetlabs/wash/cmd/util"

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
	cpuprofile := viper.GetString("cpuprofile")

	mountpoint, err := filepath.Abs(mountpoint)
	if err != nil {
		cmdutil.ErrPrintf("Could not compute the absolute path of the mountpoint %v: %v", mountpoint, err)
		return exitCode{1}
	}

	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})
	level, err := cmdutil.ParseLevel(loglevel)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	// On Ctrl-C, trigger the clean-up. This consists of shutting down the API
	// server and unmounting the FS.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	srv := server.New(mountpoint, server.Opts{
		CPUProfilePath:      cpuprofile,
		ExternalPluginsPath: externalPluginsPath,
		LogFile:             logfile,
		LogLevel:            level,
	})
	if err := srv.Start(); err != nil {
		log.Warn(err)
		return exitCode{1}
	}
	srv.Wait(sigCh)
	return exitCode{0}
}
