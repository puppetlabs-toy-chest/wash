package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/puppetlabs/wash/cmd/internal/server"
	"github.com/puppetlabs/wash/cmd/internal/shell"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/cmd/version"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// These *Flag variables are defined in root.go
var rootCommandFlag string
var rootVersionFlag bool
var rootVerifyInstallFlag bool

// Start the Wash server then present the default system shell. The server will be running in the
// current process, while the shell will be in a separate child process. We'd like the server to be
// able to prompt for input without interupting the shell (as the controller of the terminal) so
// that plugins can prompt the user for input like e.g. security tokens.
//
// To allow prompts, we start the shell process then put the Wash server (daemon) in a new session
// with `setsid`. As a new session, the daemon has a different controlling terminal and can
// therefore prompt for input without having to control the shell's terminal. This approach was
// inspired by https://blog.nelhage.com/2011/02/changing-ctty/.
//
// On exit, stop the server and return any errors.
func rootMain(cmd *cobra.Command, args []string) exitCode {
	if rootVersionFlag {
		cmdutil.Println(version.BuildVersion)
		return exitCode{0}
	}

	// Configure logrus to emit simple text
	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})

	cachedir, ok := makeCacheDir()
	if !ok {
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

	socketpath := filepath.Join(rundir, "api.sock")

	if rootVerifyInstallFlag {
		srv := server.ForVerifyInstall(mountpath, socketpath)
		if err := srv.Start(); err != nil {
			cmdutil.ErrPrintf("Install verification failed: %v\n", err)
			return exitCode{1}
		}
		srv.Stop()
		return exitCode{0}
	}

	var execfile string
	if len(args) > 0 {
		execfile = args[0]
	}

	// Set plugin interactivity to false if execfile or rootCommandFlag were specified.
	plugin.InitInteractive(execfile == "" && rootCommandFlag == "")

	// If interactive and this process is in its own process group, then fork and run the original
	// command as a new process that's not the leader of its process group. This is specifically to
	// work with shells (zsh) that try to use their parent's process group, which would no longer be
	// in the same session. By forking, we keep the original process group in its original session so
	// the child shell can still modify it.
	if plugin.IsInteractive() && os.Getpid() == unix.Getpgrp() {
		comm := exec.Command(os.Args[0], os.Args[1:]...)
		comm.Stdin = os.Stdin
		comm.Stdout = os.Stdout
		comm.Stderr = os.Stderr
		if err := comm.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return exitCode{exitErr.ExitCode()}
			}
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		return exitCode{0}
	}

	// TODO: instead of running a server in-process, can we start one in a separate process that can
	//       be shared between multiple invocations of `wash`?
	plugins, serverOpts, err := serverOptsFor(cmd)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	srv := server.New(mountpath, socketpath, plugins, serverOpts)
	if err := srv.Start(); err != nil {
		cmdutil.ErrPrintf("Unable to start server: %v\n", err)
		return exitCode{1}
	}
	defer srv.Stop()

	if plugin.IsInteractive() {
		cmdutil.Println(`Welcome to Wash!
  Wash includes several built-in commands: wexec, find, list, meta, tail.
  See commands run with wash via 'whistory', and logs with 'whistory <id>'.
Try 'help'`)
	}

	if !symlinkWash(rundir) {
		return exitCode{1}
	}

	subc := flattenSubcommands(cmd.Commands())
	comm, err := shell.Get().Command(subc, rundir)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

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
	if comm.Env == nil {
		comm.Env = os.Environ()
	}
	comm.Env = append(comm.Env,
		"WASH_SOCKET="+socketpath,
		"W="+mountpath,
		"PATH="+rundir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	comm.Dir = mountpath

	if startErr := comm.Start(); startErr != nil {
		cmdutil.ErrPrintf("%v\n", startErr)
		return exitCode{1}
	}

	// If interactive (when we might prompt the user for input, such as security tokens), create a
	// new session. If not interactive, calling setsid is pointless and might fail.
	if plugin.IsInteractive() {
		if _, err := unix.Setsid(); err != nil {
			cmdutil.ErrPrintf("Error moving Wash daemon to new session: %v", err)

			if err := comm.Process.Kill(); err != nil {
				cmdutil.ErrPrintf("Couldn't stop child shell: %v\n", err)
			}
			return exitCode{1}
		}
	}

	var exit exitCode
	if runErr := comm.Wait(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exit.value = exitErr.ExitCode()
		} else {
			cmdutil.ErrPrintf("%v\n", runErr)
			exit.value = 1
		}
	}

	if plugin.IsInteractive() {
		cmdutil.Println("Goodbye!")
	}
	return exit
}

func makeCacheDir() (cachedir string, ok bool) {
	var err error
	if cachedir, err = os.UserCacheDir(); err != nil {
		cmdutil.ErrPrintf("Unable to get user cache dir: %v\n", err)
		return
	}
	cachedir = filepath.Join(cachedir, "wash")

	// ensure cache directory exists
	if err = os.MkdirAll(cachedir, 0750); err != nil {
		cmdutil.ErrPrintf("Unable to create cache dir %v: %v\n", cachedir, err)
		return
	}
	ok = true
	return
}

func flattenSubcommands(subcommands []*cobra.Command) []string {
	// Executable file can't override shell built-ins, so use wexec instead of exec.
	// List also isn't very feature complete so we don't override ls.
	var subc []string
	for _, subcommand := range subcommands {
		tokens := strings.SplitN(subcommand.Use, " ", 2)
		if len(tokens) < 1 {
			panic("all subcommands should have non-empty usage")
		}
		name := tokens[0]
		// Specifically skip server as undocumented when running in wash shell.
		if name == "server" {
			continue
		}

		var aliases []string
		for _, alias := range subcommand.Aliases {
			if strings.HasPrefix(alias, "w") {
				aliases = append(aliases, alias)
			}
		}

		if len(aliases) == 0 {
			aliases = append(aliases, name)
		}
		subc = append(subc, aliases...)
	}
	return subc
}

func symlinkWash(rundir string) (ok bool) {
	washPath, err := os.Executable()
	if err != nil {
		cmdutil.ErrPrintf("Error finding wash executable: %v\n", err)
		return
	}

	newWashPath := filepath.Join(rundir, "wash")
	if err := os.Symlink(washPath, newWashPath); err != nil {
		cmdutil.ErrPrintf("Error linking wash executable to %v: %v\n", newWashPath, err)
		return
	}

	ok = true
	return
}
