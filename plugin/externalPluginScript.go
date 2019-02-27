package plugin

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
)

// ExternalPluginScript represents an external plugin's script
type ExternalPluginScript struct {
	Path string
}

// InvokeAndWait shells out to the plugin script, passing it the given
// arguments. It waits for the script to exit, then returns its standard
// output.
//
// TODO: Add a suitable timeout. This could be done w/ CommandContext per
// https://golang.org/pkg/os/exec/#Cmd.Wait. Could this be specified by
// plugin authors in the top-level YAML file? Should it be a per-entry
// thing?
func (s ExternalPluginScript) InvokeAndWait(args ...string) ([]byte, error) {
	log.Debugf("Running command: %v %v", s.Path, strings.Join(args, " "))

	cmd := exec.Command(s.Path, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		ws := exitErr.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	} else if err != nil {
		return nil, err
	}

	stdout := stdoutBuf.Bytes()
	stderr := stderrBuf.Bytes()
	if exitCode == 0 {
		if len(stderr) != 0 {
			log.Debugf("stderr: %v", string(stderr))
		}
	} else {
		// TODO: Wrap standard error into a structured Wash error
		return nil, fmt.Errorf("script returned a non-zero exit code of %v. stderr output: %v", exitCode, string(stderr))
	}

	return stdout, nil
}
