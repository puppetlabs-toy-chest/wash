package shell

import (
	"os"
	"os/exec"
	"path/filepath"
)

type bash struct {
	sh string
}

func (b bash) Command(subcommands []string, rundir string) (*exec.Cmd, error) {
	// Generate and invoke custom .bashenv and .bashrc files.
	// - .bashenv will alias subcommands, then load ~/.washenv (if present).
	// - .bashrc will load ~/.bashrc (if ~/.washrc is absent), then configure the prompt,
	//   then load ~/.washrc (if present).

	envpath := filepath.Join(rundir, ".bashenv")
	rcpath := filepath.Join(rundir, ".bashrc")

	cmd := exec.Command(b.sh, "--rcfile", rcpath)
	cmd.Env = append(os.Environ(), "BASH_ENV="+envpath)

	bashenv, err := os.Create(envpath)
	if err != nil {
		return nil, err
	}
	defer bashenv.Close()

	if env := os.Getenv("BASH_ENV"); env != "" {
		_, err = bashenv.WriteString("[[ -s '~/" + env + "' && ! -s ~/.washenv ]] && source '" + env + "'\n")
		if err != nil {
			return nil, err
		}
	}
	for _, alias := range subcommands {
		_, err = bashenv.WriteString("alias " + alias + "='WASH_EMBEDDED=1 wash " + alias + "'\n")
		if err != nil {
			return nil, err
		}
	}
	_, err = bashenv.WriteString("[[ -s ~/.washenv ]] && source ~/.washenv\n")
	if err != nil {
		return nil, err
	}

	bashrc, err := os.Create(rcpath)
	if err != nil {
		return nil, err
	}
	defer bashrc.Close()

	_, err = bashrc.WriteString(`source ` + envpath + `
[[ -s ~/.bashrc && ! -s ~/.washrc ]] && source ~/.bashrc

WASH_BASE=$(pwd)
function prompter() {
	export PS1="\e[0;36mwash $(realpath --relative-to=$WASH_BASE $(pwd))\e[0;32m ❯\e[m "
}
export PROMPT_COMMAND=prompter

[[ -s ~/.washrc ]] && source ~/.washrc
`)
	return cmd, err
}
