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
	if err := writeAliases(subcommands, rundir); err != nil {
		return nil, err
	}

	return exec.Command(b.sh), nil
}

// Helper to write executable wrappers for available Wash subcommands based on their aliases.
// This is broadly useful because things like `xargs` ignore builtins and aliases.
func writeAliases(subcommands []string, rundir string) error {
	for _, alias := range subcommands {
		if err := writeAlias(filepath.Join(rundir, alias), alias); err != nil {
			return fmt.Errorf("unable to create alias for subcommand %v: %v", alias, err)
		}
	}
	return nil
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
