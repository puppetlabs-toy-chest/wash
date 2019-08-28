package shell

import (
	"io/ioutil"
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

	var common, content string
	for _, alias := range subcommands {
		common += "alias " + alias + "='WASH_EMBEDDED=1 wash " + alias + "'\n"
	}

	if env := os.Getenv("BASH_ENV"); env != "" {
		content += "[[ -s '~/" + env + "' && ! -s ~/.washenv ]] && source '" + env + "'\n"
	}
	content += common
	content += "[[ -s ~/.washenv ]] && source ~/.washenv\n"
	if err := ioutil.WriteFile(envpath, []byte(content), 0644); err != nil {
		return nil, err
	}

	content = `source ` + envpath + `
[[ -s ~/.bashrc && ! -s ~/.washrc ]] && source ~/.bashrc
`
	// Re-add aliases in case .bashrc overrode them.
	content += common
	content += `
function prompter() {
	export PS1="\e[0;36mwash $(realpath --relative-to=$W $(pwd))\e[0;32m ❯\e[m "
}
export PROMPT_COMMAND=prompter

[[ -s ~/.washrc ]] && source ~/.washrc
`
	if err := ioutil.WriteFile(rcpath, []byte(content), 0644); err != nil {
		return nil, err
	}
	return cmd, nil
}
