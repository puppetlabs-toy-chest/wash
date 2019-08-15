package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// basic implements a shell with no extra setup. Wash commands are implemented as scripts.
type basic struct {
	sh string
}

func (b basic) Command(subcommands []string, rundir string) (*exec.Cmd, error) {
	// These are executables instead of aliases because putting alias declarations at the beginning
	// of stdin for the command doesn't work right (issues with TTY).
	// Generate executable wrappers for available subcommands based on their aliases.
	for _, alias := range subcommands {
		if err := writeAlias(filepath.Join(rundir, alias), alias); err != nil {
			return nil, fmt.Errorf("unable to create alias for subcommand %v: %v", alias, err)
		}
	}

	return exec.Command(b.sh), nil
}

// Create an executable file at the given path that invokes the given wash subcommand.
func writeAlias(path, subcommand string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0750)
	if err != nil {
		return err
	}

	// Use an Embedded config option to specialize help for the wash shell environment.
	_, err = f.WriteString("#!/bin/sh\nWASH_EMBEDDED=1 exec wash " + subcommand + " \"$@\"")
	f.Close()
	return err
}
