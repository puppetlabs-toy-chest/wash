package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/puppetlabs/wash/journal"
)

// ExternalPluginScript represents an external plugin's script
type ExternalPluginScript interface {
	Path() string
	InvokeAndWait(ctx context.Context, args ...string) ([]byte, error)
}

type externalPluginScript struct {
	path string
}

// NewExternalPluginScript returns a new external plugin script
// object
func NewExternalPluginScript(path string) ExternalPluginScript {
	return externalPluginScript{path: path}
}

func (s externalPluginScript) Path() string {
	return s.path
}

// InvokeAndWait shells out to the plugin script, passing it the given
// arguments. It waits for the script to exit, then returns its standard
// output.
//
// TODO: Add a suitable timeout. This could be done w/ CommandContext per
// https://golang.org/pkg/os/exec/#Cmd.Wait. Could this be specified by
// plugin authors in the top-level YAML file? Should it be a per-entry
// thing?
func (s externalPluginScript) InvokeAndWait(ctx context.Context, args ...string) ([]byte, error) {
	journal.Record(ctx, "Running command: %v %v", s.path, strings.Join(args, " "))

	cmd := exec.Command(s.path, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	exitCode, err := ExitCodeFromErr(err)
	if err != nil {
		return nil, err
	}

	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()
	if exitCode == 0 {
		if len(stderr) != 0 {
			journal.Record(ctx, "stderr: %v", string(stderr))
		}
	} else {
		// TODO: Wrap standard error into a structured Wash error
		return nil, fmt.Errorf("script returned a non-zero exit code of %v. stderr output: %v", exitCode, string(stderr))
	}

	return stdout, nil
}
