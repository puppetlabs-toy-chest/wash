package shell

import (
	"os"
	"os/exec"
)

// Shell provides setup for a specific shell, such as bash or zsh.
type Shell interface {
	// Command constructs the command to invoke the shell. Subcommands should be made available in the
	// shell environment as `env WASH_EMBEDDED=1 wash <subcommand>`. Rundir is a temporary directory
	// that will be added to PATH when the command is invoked; you can use it to add new executables
	// or store other temporary files.
	Command(subcommands []string, rundir string) (*exec.Cmd, error)
}

// Get returns an implementation for the shell described by the SHELL environment variable.
func Get() Shell {
	// Run the default system shell.
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "/bin/sh"
	}

	return basic{sh: sh}
}
