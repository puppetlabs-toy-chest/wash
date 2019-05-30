package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/server"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCommandFlag is associated with the `-c` option of the root command, set in root.go.
var rootCommandFlag string

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

func runShell(rundir, mountpath, socketpath, execfile string) exitCode {
	// Add rundir to the PATH and put aliases there.
	washPath, err := os.Executable()
	if err != nil {
		cmdutil.ErrPrintf("Error finding wash executable: %v\n", err)
		return exitCode{1}
	}

	newWashPath := filepath.Join(rundir, "wash")
	if err := os.Symlink(washPath, newWashPath); err != nil {
		cmdutil.ErrPrintf("Error linking wash executable to %v: %v\n", newWashPath, err)
		return exitCode{1}
	}
	// Executable file can't override shell built-ins, so use wexec instead of exec.
	// List also isn't very feature complete so we don't override ls.
	// These are executables instead of aliases because putting alias declarations at the beginning
	// of stdin for the command doesn't work right.
	aliases := map[string]string{
		"wclear":   "clear",
		"wexec":    "exec",
		"find":     "find",
		"help":     "help",
		"winfo":    "info",
		"whistory": "history",
		"list":     "list",
		"meta":     "meta",
		"tail":     "tail",
	}
	for name, subcommand := range aliases {
		if err := writeAlias(filepath.Join(rundir, name), subcommand); err != nil {
			cmdutil.ErrPrintf("Error creating alias %v for subcommand %v: %v\n", name, subcommand, err)
			return exitCode{1}
		}
	}

	pathEnv := os.Getenv("PATH")
	if err := os.Setenv("PATH", rundir+string(os.PathListSeparator)+pathEnv); err != nil {
		cmdutil.ErrPrintf("Error adding wash executables to PATH: %v\n", err)
		return exitCode{1}
	}

	// Run the default system shell.
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}

	comm := exec.Command(sh)
	if execfile != "" {
		file, err := os.Open(execfile)
		if err != nil {
			cmdutil.ErrPrintf("Error reading file %v: %v\n", execfile, err)
			return exitCode{1}
		}
		comm.Stdin = file
		defer file.Close()
	} else if rootCommandFlag != "" {
		comm.Stdin = strings.NewReader(rootCommandFlag)
	} else {
		comm.Stdin = os.Stdin
	}
	comm.Stdout = os.Stdout
	comm.Stderr = os.Stderr
	comm.Env = append(os.Environ(), "WASH_SOCKET="+socketpath)
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

	// Mountpath is not cleaned up correctly if removed as part of deleting rundir, so it's placed
	// in a separate location. The server has reported that it's completely done by the time we
	// delete rundir, so I'm not sure why it doesn't clean up correctly. Alternatively, adding a
	// 10ms sleep after srv.Stop() seemed to let it successfully unmount (with OSXFUSE).
	mountpath, err := ioutil.TempDir(cachedir, "mnt")
	if err != nil {
		cmdutil.ErrPrintf("Unable to create temporary mountpoint in %v: %v\n", cachedir, err)
		return exitCode{1}
	}
	defer os.RemoveAll(mountpath)

	// Create a temporary run space for aliases and server files.
	rundir, err := ioutil.TempDir(cachedir, "run")
	if err != nil {
		cmdutil.ErrPrintf("Error creating temporary run location in %v: %v\n", cachedir, err)
		return exitCode{1}
	}
	defer os.RemoveAll(rundir)

	var execfile string
	if len(args) > 0 {
		execfile = args[0]
	}

	// Set plugin interactivity to false if execfile or rootCommandFlag were specified.
	plugin.InitInteractive(execfile == "" && rootCommandFlag == "")

	// TODO: instead of running a server in-process, can we start one in a separate process that can
	//       be shared between multiple invocations of `wash`?
	serverOpts, err := serverOptsFor(cmd)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	socketpath := filepath.Join(rundir, "api.sock")
	srv := server.New(mountpath, socketpath, serverOpts)
	if err := srv.Start(); err != nil {
		cmdutil.ErrPrintf("Unable to start server: %v\n", err)
		return exitCode{1}
	}

	if plugin.IsInteractive() {
		fmt.Println(`Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'`)
	}

	exit := runShell(rundir, mountpath, socketpath, execfile)

	srv.Stop()
	if plugin.IsInteractive() {
		fmt.Println("Goodbye!")
	}
	return exit
}
