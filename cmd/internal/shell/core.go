package shell

import (
	"os"
	"os/exec"
	"path/filepath"
)

// Shell provides setup for a specific shell, such as bash or zsh.
type Shell interface {
	// Command constructs the command to invoke the shell. Subcommands should be made available in the
	// shell environment as `env WASH_EMBEDDED=1 wash <subcommand>`. Rundir is a temporary directory
	// that will be added to `PATH` when the command is invoked; you can use it to add new executables
	// or store other temporary files. A `W` environment will also be set to the path where the shell
	// starts.
	//
	// Implementations should support their native interactive and non-interactive config, as well as
	// Wash's (.washrc and .washenv, respectively). They should:
	//   1. if ~/.washenv does not exist, load the shell's default non-interactive config
	//   1. configure subcommand aliases
	//   1. if ~/.washenv exists, load it
	// Additionally for interactive invocations they should:
	//   1. if ~/.washrc does not exist, load the shell's default interactive config
	//   1. configure the prompt
	//   1. if ~/.washrc exists, load it
	Command(subcommands []string, rundir string) (*exec.Cmd, error)
}

// Get returns an implementation for the shell described by the SHELL environment variable.
func Get() Shell {
	switch sh := os.Getenv("SHELL"); filepath.Base(sh) {
	case "bash":
		return bash{sh: sh}
	case "zsh":
		return zsh{sh: sh}
	default:
		// Basic is a fallback that doesn't fully implement the Shell semantics. It provides the common
		// subset we can expect from a Bourne Shell.
		if sh == "" {
			sh = "/bin/sh"
		}
		return basic{sh: sh}
	}
}
