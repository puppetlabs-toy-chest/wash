package shell

import (
	"io/ioutil"
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
	// Override ZDOTDIR so zsh looks for our configs, but save the original ZDOTDIR so we can use it
	// when loading zsh-specific config that may rely on it being set.
	cmd.Env = append(os.Environ(), "ZDOTDIR="+rundir)
	zdotdir := os.Getenv("ZDOTDIR")

	var common string
	for _, alias := range subcommands {
		common += "alias " + alias + "='WASH_EMBEDDED=1 wash " + alias + "'\n"
	}

	content := `if [[ ! -s ~/.washenv ]]; then
	# Reset ZDOTDIR for zsh config, then set it back so we load Wash's zshrc
  ZDOTDIR='` + zdotdir + `'
  if [[ -s "${ZDOTDIR:-$HOME}/.zshenv" ]]; then
    source "${ZDOTDIR:-$HOME}/.zshenv"
	fi
	ZDOTDIR='` + rundir + `'
fi
`
	content += common
	content += "if [[ -s ~/.washenv ]]; then source ~/.washenv; fi\n"
	if err := ioutil.WriteFile(filepath.Join(rundir, ".zshenv"), []byte(content), 0644); err != nil {
		return nil, err
	}

	content = `if [[ ! -s ~/.washrc ]]; then
	ZDOTDIR='` + zdotdir + `'
  if [[ -s "${ZDOTDIR:-$HOME}/.zprofile" ]]; then source "${ZDOTDIR:-$HOME}/.zprofile"; fi
  if [[ -s "${ZDOTDIR:-$HOME}/.zshrc" ]]; then source "${ZDOTDIR:-$HOME}/.zshrc"; fi
fi
`
	// Re-add aliases in case .zprofile or .zshrc overrode them.
	content += common
	content += `
function prompter() {
  local prompt_path
  if [ -x "$(command -v realpath)" ]; then
    prompt_path=$(realpath --relative-to=$W $(pwd))
  else
    prompt_path=$(basename $(pwd))
  fi
  PROMPT="%F{cyan}wash ${prompt_path}%F{green} ❯%f "
}

autoload -Uz add-zsh-hook
add-zsh-hook precmd prompter

if [[ -s ~/.washrc ]]; then source ~/.washrc; fi
`
	if err := ioutil.WriteFile(filepath.Join(rundir, ".zshrc"), []byte(content), 0644); err != nil {
		return nil, err
	}
	return cmd, nil
}
