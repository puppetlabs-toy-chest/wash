package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/puppetlabs/wash/cmd/internal/server"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Create an executable file at the given path that invokes the given wash subcommand.
func writeAlias(path, subcommand string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}
	_, err = f.WriteString("#!/bin/sh\nexec wash " + subcommand + " \"$@\"")
	f.Close()
	return err
}

func runShell(cachedir, mountpath string) exitCode {
	// Create a temporary run space with wash and aliases. Add it to the PATH.
	runpath, err := ioutil.TempDir(cachedir, "run")
	if err != nil {
		cmdutil.ErrPrintf("Error creating temporary run location in %v: %v\n", cachedir, err)
		return exitCode{1}
	}
	defer os.RemoveAll(runpath)

	washPath, err := os.Executable()
	if err != nil {
		cmdutil.ErrPrintf("Error finding wash executable: %v\n", err)
		return exitCode{1}
	}

	newWashPath := filepath.Join(runpath, "wash")
	if err := os.Symlink(washPath, newWashPath); err != nil {
		cmdutil.ErrPrintf("Error linking wash executable to %v: %v\n", newWashPath, err)
		return exitCode{1}
	}
	// Executable file can't override shell built-ins, so use wexec instead of exec.
	// List also isn't very feature complete so we don't override ls.
	// These are executables instead of aliases because putting alias declarations at the beginning
	// of stdin for the command doesn't work right.
	aliases := map[string]string{
		"wclear":  "clear",
		"wexec":   "exec",
		"find":    "find",
		"history": "history",
		"list":    "list",
		"meta":    "meta",
		"tail":    "tail",
	}
	for name, subcommand := range aliases {
		if err := writeAlias(filepath.Join(runpath, name), subcommand); err != nil {
			cmdutil.ErrPrintf("Error creating alias %v for subcommand %v: %v\n", name, subcommand, err)
			return exitCode{1}
		}
	}

	pathEnv := os.Getenv("PATH")
	if err := os.Setenv("PATH", runpath+string(os.PathListSeparator)+pathEnv); err != nil {
		cmdutil.ErrPrintf("Error adding wash executables to PATH: %v\n", err)
		return exitCode{1}
	}

	// Run the default system shell.
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}

	comm := exec.Command(sh)
	comm.Stdin = os.Stdin
	comm.Stdout = os.Stdout
	comm.Stderr = os.Stderr
	comm.Dir = mountpath
	if runErr := comm.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			return exitCode{exitErr.ExitCode()}
		}
		cmdutil.ErrPrintf("%v\n", runErr)
		return exitCode{1}
	}
	return exitCode{0}
}

// Start the wash server, then present the default system shell.
// On exit, stop the server and return any errors.
func rootMain(cmd *cobra.Command, args []string) exitCode {
	loglevel := viper.GetString("loglevel")
	logfile := viper.GetString("logfile")

	level, err := cmdutil.ParseLevel(loglevel)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	// Configure logrus to emit simple text
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})

	cachedir, err := os.UserCacheDir()
	if err != nil {
		cmdutil.ErrPrintf("Unable to get user cache dir: %v\n", err)
		return exitCode{1}
	}
	cachedir = filepath.Join(cachedir, "wash")

	// ensure cache directory exists
	if err = os.MkdirAll(cachedir, 0750); err != nil {
		cmdutil.ErrPrintf("Unable to create cache dir %v: %v\n", cachedir, err)
		return exitCode{1}
	}

	mountpath, err := ioutil.TempDir(cachedir, "mnt")
	if err != nil {
		cmdutil.ErrPrintf("Unable to create temporary mountpoint in %v: %v\n", cachedir, err)
		return exitCode{1}
	}
	defer os.RemoveAll(mountpath)

	// TODO: instead of running a server in-process, can we start one in a separate process that can
	//       be shared between multiple invocations of `wash`?
	srv := server.New(mountpath, server.Opts{LogFile: logfile, LogLevel: level})
	if err := srv.Start(); err != nil {
		cmdutil.ErrPrintf("Unable to start server: %v\n", err)
		return exitCode{1}
	}

	fmt.Println(`Welcome to Wash!
  Wash includes several built-in commands: wclear, wexec, find, list, meta, tail.
  Commands run with wash can be seen via 'history', and logs for those commands with 'history <id>'.`)

	exit := runShell(cachedir, mountpath)

	srv.Stop()
	fmt.Println("Goodbye!")
	return exit
}
