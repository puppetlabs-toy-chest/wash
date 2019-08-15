package shell

import (
	"os"
	"os/exec"
	"path/filepath"
)

type zsh struct {
	sh string
}

func (z zsh) Command(subcommands []string, rundir string) (*exec.Cmd, error) {
	// Generate and invoke custom .zshenv and .zshrc files.
	// - .zshenv will load ~/.zshenv (if ~/.washenv is absent), then alias subcommands,
	//   then load ~/.washenv (if present).
	// - .zshrc will load ~/.zshrc (if ~/.washrc is absent), then configure the prompt,
	//   then load ~/.washrc (if present).

	cmd := exec.Command(z.sh)
	cmd.Env = append(os.Environ(), "ZDOTDIR="+rundir)

	zshenv, err := os.Create(filepath.Join(rundir, ".zshenv"))
	if err != nil {
		return nil, err
	}
	defer zshenv.Close()

	_, err = zshenv.WriteString("[[ -s ~/.zshenv && ! -s ~/.washenv ]] && source ~/.zshenv\n")
	if err != nil {
		return nil, err
	}
	for _, alias := range subcommands {
		_, err = zshenv.WriteString("alias " + alias + "='WASH_EMBEDDED=1 wash " + alias + "'\n")
		if err != nil {
			return nil, err
		}
	}
	_, err = zshenv.WriteString("[[ -s ~/.washenv ]] && source ~/.washenv\n")
	if err != nil {
		return nil, err
	}

	zshrc, err := os.Create(filepath.Join(rundir, ".zshrc"))
	if err != nil {
		return nil, err
	}
	defer zshrc.Close()

	_, err = zshrc.WriteString(`source $ZDOTDIR/.zshenv
[[ -s ~/.zshrc && ! -s ~/.washrc ]] && source ~/.zshrc

WASH_BASE=$(pwd)
function prompter() {
	PROMPT="%F{cyan}wash $(realpath --relative-to=$WASH_BASE $(pwd))%F{green} ❯%f "
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd prompter

[[ -s ~/.washrc ]] && source ~/.washrc
`)
	return cmd, err
}
