package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/puppetlabs/wash/activity"
)

// externalPluginScript represents an external plugin's script
type externalPluginScript interface {
	Path() string
	InvokeAndWait(ctx context.Context, args ...string) ([]byte, error)
}

type externalPluginScriptImpl struct {
	path string
}

func (s externalPluginScriptImpl) Path() string {
	return s.path
}

// InvokeAndWait shells out to the plugin script, passing it the given
// arguments. It waits for the script to exit, then returns its standard
// output.
func (s externalPluginScriptImpl) InvokeAndWait(ctx context.Context, args ...string) ([]byte, error) {
	activity.Record(ctx, "Running command: %v %v", s.Path(), strings.Join(args, " "))

	// Use CommandContext to ensure that the process is killed when the context
	// is cancelled
	cmd := exec.CommandContext(ctx, s.Path(), args...)

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
			activity.Record(ctx, "stderr: %v", string(stderr))
		}
	} else {
		// TODO: Wrap standard error into a structured Wash error
		return nil, fmt.Errorf("script returned a non-zero exit code of %v. stderr output: %v", exitCode, string(stderr))
	}

	return stdout, nil
}
