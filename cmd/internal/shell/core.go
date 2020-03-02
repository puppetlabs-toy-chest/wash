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
	//   1. reconfigure subcommand aliases (in case they were overridden)
	//   1. configure the prompt to show your location within the Wash hierarchy (use preparePrompt)
	//   1. override cd so `cd` without arguments changes directory to $W (use overrideCd)
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

// Create the declaration for a `prompter` function that generates the prompt
//   `%F{cyan}wash ${prompt_path}%F{green} ❯%f `
// with substitution for shell-specific portions of the function.
func preparePrompt(cyan, green, reset, assign string) string {
	return `
function prompter() {
	local prompt_path
	if [ -x "$(command -v realpath)" ]; then
		prompt_path=$(realpath --relative-base=$W "$(pwd)")
	else
		prompt_path=$(basename "$(pwd)")
	fi
	` + assign + `="` + cyan + `wash ${prompt_path}` + green + ` ❯` + reset + ` "
}
`
}

// Create the declaration for a `cd` function that returns to the Wash root when no arguments are
// supplied.
func overrideCd() string {
	return `
function cd { if (( $# == 0 )); then builtin cd $W; else builtin cd $*; fi }
`
}
