package cmd

import (
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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

	// When Wash prompts for input but it doesn't currently control STDIN, it will get SIGTTOU. This
	// is common when a plugin prompts for input in response to running a command like `ls` or `cat`.
	// Ignore this signal so we aren't suspended when prompting for input from the background.
	// We also ignore SIGTTIN because http://curiousthing.org/sigttin-sigttou-deep-dive-linux
	// suggests that's the signal we should be getting, even though I haven't seen it in testing.
	signal.Ignore(syscall.SIGTTOU, syscall.SIGTTIN)

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
		"PATH="+rundir+string(os.PathListSeparator)+os.Getenv("PATH"),
	)
	comm.Dir = mountpath

	var exit exitCode
	if runErr := comm.Run(); runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exit.value = exitErr.ExitCode()
		}
		cmdutil.ErrPrintf("%v\n", runErr)
		exit.value = 1
	}

	srv.Stop()
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
