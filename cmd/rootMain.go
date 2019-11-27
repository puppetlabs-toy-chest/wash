package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/puppetlabs/wash/cmd/internal/server"
	"github.com/puppetlabs/wash/cmd/internal/shell"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCommandFlag is associated with the `-c` option of the root command, set in root.go.
var rootCommandFlag string

// Start the wash server, then present the default system shell.
// On exit, stop the server and return any errors.
func rootMain(cmd *cobra.Command, args []string) exitCode {
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

	var execfile string
	if len(args) > 0 {
		execfile = args[0]
	}

	// Set plugin interactivity to false if execfile or rootCommandFlag were specified.
	plugin.InitInteractive(execfile == "" && rootCommandFlag == "")

	// TODO: instead of running a server in-process, can we start one in a separate process that can
	//       be shared between multiple invocations of `wash`?
	plugins, serverOpts, err := serverOptsFor(cmd)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	socketpath := filepath.Join(rundir, "api.sock")
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

	// Inspired by https://blog.nelhage.com/2011/02/changing-ctty/. After the child shell has
	// started, we re-parent the Wash daemon to be in a child session of the shell so that when Wash
	// prompts for input it's within the TTY of that terminal.
	if startErr := comm.Start(); startErr != nil {
		cmdutil.ErrPrintf("%v\n", startErr)
		return exitCode{1}
	}

	codeCh := make(chan int)
	go func() {
		if runErr := comm.Wait(); runErr != nil {
			if exitErr, ok := runErr.(*exec.ExitError); ok {
				codeCh <- exitErr.ExitCode()
			} else {
				cmdutil.SafeErrPrintf("%v\n", runErr)
				codeCh <- 1
			}
		} else {
			codeCh <- 0
		}
		close(codeCh)
	}()

	var exit exitCode
	killShellProcess := func() exitCode {
		if err := comm.Process.Kill(); err != nil {
			cmdutil.SafeErrPrintf("Couldn't stop child shell: %v\n", err)
		}
		return exitCode{1}
	}

	// The new shell will initially be part of our process group. We wait for it to become the leader
	// of its own process group, then move Wash to be a new session under that process group.
	washPid := os.Getpid()
	pid := comm.Process.Pid
	for {
		time.Sleep(10 * time.Millisecond)

		select {
		case code := <-codeCh:
			// Child shell stopped, so just exit with it's exit code.
			exit.value = code
			break
		default:
			// Fall-through
		}

		pgid, err := unix.Getpgid(pid)
		if err != nil {
			cmdutil.SafeErrPrintf("Error moving Wash daemon to new shell: %v\n", err)
			return killShellProcess()
		}

		// Once the shell is the leader of its own process group, move Wash to that group and
		// put it in a new session.
		if pgid == pid {
			if err := unix.Setpgid(washPid, pgid); err != nil {
				cmdutil.SafeErrPrintf("Error moving Wash daemon to new shell: %v\n", err)
				return killShellProcess()
			}

			if _, err := unix.Setsid(); err != nil {
				cmdutil.SafeErrPrintf("Error starting new session for Wash daemon: %v\n", err)
				return killShellProcess()
			}

			break
		}
	}

	if code, ok := <-codeCh; ok {
		exit.value = code
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
