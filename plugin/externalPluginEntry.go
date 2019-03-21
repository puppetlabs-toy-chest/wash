package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/puppetlabs/wash/journal"
)

type decodedAttributes struct {
	// Atime, Mtime, and Ctime are in Unix time
	Atime int64         `json:"atime"`
	Mtime int64         `json:"mtime"`
	Ctime int64         `json:"ctime"`
	Mode  interface{}   `json:"mode"`
	Size  uint64        `json:"size"`
	Valid time.Duration `json:"valid"`
}

func unixSecondsToTimeAttr(seconds int64) time.Time {
	if seconds <= 0 {
		return time.Time{}
	}

	return time.Unix(seconds, 0)
}

func (a decodedAttributes) toAttributes() (Attributes, error) {
	attr := Attributes{
		Atime: unixSecondsToTimeAttr(a.Atime),
		Mtime: unixSecondsToTimeAttr(a.Mtime),
		Ctime: unixSecondsToTimeAttr(a.Ctime),
		Size:  a.Size,
		Valid: a.Valid,
	}

	if a.Mode != nil {
		mode, err := ToFileMode(a.Mode)
		if err != nil {
			return Attributes{}, err
		}
		attr.Mode = mode
	}

	return attr, nil
}

type decodedCacheTTLs struct {
	List     time.Duration `json:"list"`
	Open     time.Duration `json:"open"`
	Metadata time.Duration `json:"metadata"`
}

// decodedExternalPluginEntry describes a decoded serialized entry.
type decodedExternalPluginEntry struct {
	Name             string            `json:"name"`
	SupportedActions []string          `json:"supported_actions"`
	CacheTTLs        decodedCacheTTLs  `json:"cache_ttls"`
	Attributes       decodedAttributes `json:"attributes"`
	State            string            `json:"state"`
}

func (e decodedExternalPluginEntry) toExternalPluginEntry() (*externalPluginEntry, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("the entry name must be provided")
	}
	if e.SupportedActions == nil {
		return nil, fmt.Errorf("the entry's supported actions must be provided")
	}

	attr, err := e.Attributes.toAttributes()
	if err != nil {
		return nil, err
	}

	entry := &externalPluginEntry{
		EntryBase:        NewEntry(e.Name),
		supportedActions: e.SupportedActions,
		state:            e.State,
		attr:             attr,
	}
	entry.setCacheTTLs(e.CacheTTLs)

	return entry, nil
}

// externalPluginEntry represents an entry from an external plugin. It consists
// of its name, its object (as serialized JSON), and the path to its
// main plugin script.
type externalPluginEntry struct {
	EntryBase
	script           externalPluginScript
	supportedActions []string
	state            string
	attr             Attributes
}

func (e *externalPluginEntry) setCacheTTLs(ttls decodedCacheTTLs) {
	if ttls.List != 0 {
		e.SetTTLOf(List, ttls.List*time.Second)
	}
	if ttls.Open != 0 {
		e.SetTTLOf(Open, ttls.Open*time.Second)
	}
	if ttls.Metadata != 0 {
		e.SetTTLOf(Metadata, ttls.Metadata*time.Second)
	}
}

// List lists the entry's children, if it has any.
func (e *externalPluginEntry) List(ctx context.Context) ([]Entry, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "list", e.id(), e.state)
	if err != nil {
		return nil, err
	}

	var decodedEntries []decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedEntries); err != nil {
		journal.Record(
			ctx,
			"could not decode the entries from stdout\nreceived:\n%v\nexpected something like:\n%v",
			strings.TrimSpace(string(stdout)),
			"[{\"name\":\"<name_of_first_entry>\",\"supported_actions\":[\"list\"]},{\"name\":\"<name_of_second_entry>\",\"supported_actions\":[\"list\"]}]",
		)

		return nil, fmt.Errorf("could not decode the entries from stdout: %v", err)
	}

	entries := make([]Entry, len(decodedEntries))
	for i, decodedExternalPluginEntry := range decodedEntries {
		entry, err := decodedExternalPluginEntry.toExternalPluginEntry()
		if err != nil {
			return nil, err
		}
		entry.script = e.script

		entries[i] = entry
	}

	return entries, nil
}

// Open returns the entry's content
func (e *externalPluginEntry) Open(ctx context.Context) (SizedReader, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "read", e.id(), e.state)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(stdout), nil
}

// Metadata displays the resource's metadata
func (e *externalPluginEntry) Metadata(ctx context.Context) (MetadataMap, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "metadata", e.id(), e.state)
	if err != nil {
		return nil, err
	}

	var metadata MetadataMap
	if err := json.Unmarshal(stdout, &metadata); err != nil {
		journal.Record(
			ctx,
			"could not decode the metadata from stdout\nreceived:\n%v\nexpected something like:\n%v",
			strings.TrimSpace(string(stdout)),
			"{\"key1\":\"value1\",\"key2\":\"value2\"}",
		)

		return nil, fmt.Errorf("could not decode the metadata from stdout: %v", err)
	}

	return metadata, nil
}

// Attr returns the entry's filesystem attributes
func (e *externalPluginEntry) Attr(ctx context.Context) (Attributes, error) {
	return e.attr, nil
}

type stdoutStreamer struct {
	stdoutRdr io.ReadCloser
	cmd       *exec.Cmd
}

func (s *stdoutStreamer) Read(p []byte) (int, error) {
	return s.stdoutRdr.Read(p)
}

func (s *stdoutStreamer) Close() error {
	var err error

	if closeErr := s.stdoutRdr.Close(); closeErr != nil {
		err = fmt.Errorf("error closing stdout: %v", closeErr)
	}

	if signalErr := s.cmd.Process.Signal(syscall.SIGTERM); signalErr != nil {
		signalErr = fmt.Errorf(
			"error sending SIGTERM to process with PID %v: %v",
			s.cmd.Process.Pid,
			signalErr,
		)

		if err != nil {
			err = fmt.Errorf("%v; and %v", err, signalErr)
		} else {
			err = signalErr
		}
	}

	return err
}

// Stream streams the entry's content
func (e *externalPluginEntry) Stream(ctx context.Context) (io.Reader, error) {
	cmd, stdoutR, stderrR, err := CreateCommand(
		e.script.Path(),
		"stream",
		e.id(),
		e.state,
	)

	if err != nil {
		return nil, err
	}

	cmdStr := fmt.Sprintf("%v %v %v %v", e.script.Path(), "stream", e.id(), e.state)

	journal.Record(ctx, "Starting command: %v", cmdStr)
	if err := cmd.Start(); err != nil {
		journal.Record(ctx, "Closed command stdout: %v", stdoutR.Close())
		journal.Record(ctx, "Closed command stderr: %v", stderrR.Close())
		return nil, err
	}

	header := "200"
	headerRdrCh := make(chan error, 1)
	go func() {
		defer close(headerRdrCh)

		headerBytes := []byte(header + "\n")
		buf := make([]byte, len(headerBytes), cap(headerBytes))

		n, err := stdoutR.Read(buf)
		if err != nil {
			headerRdrCh <- err
			return
		}
		if n != len(headerBytes) || string(buf) != string(headerBytes) {
			headerRdrCh <- fmt.Errorf("read an invalid header: %v", string(buf))
			return
		}

		// Good to go.
		headerRdrCh <- nil
	}()

	timeout := 5 * time.Second
	timer := time.After(timeout)

	// Waiting for the command to finish ensures that the
	// stdout + stderr readers are closed
	//
	// TODO: Talk about how to handle timeout here, i.e.
	// whether we should kill the command or not. Should we
	// do this in a separate goroutine?
	waitForCommandToFinish := func() {
		journal.Record(ctx, "Waiting for command: %v", cmdStr)
		_, err := ExitCodeFromErr(cmd.Wait())
		if err != nil {
			journal.Record(ctx, "Failed waiting for command: %v", err)
		}
	}

	select {
	case err := <-headerRdrCh:
		if err != nil {
			defer waitForCommandToFinish()

			// Try to get more context from stderr, if there is
			// any
			stderr, readErr := ioutil.ReadAll(stderrR)
			if readErr == nil && len(stderr) != 0 {
				err = fmt.Errorf(string(stderr))
			}

			return nil, fmt.Errorf("failed to read the header: %v", err)
		}

		// Keep reading from stderr so that the streaming isn't
		// blocked when its buffer is full.
		go func() {
			defer func() {
				journal.Record(ctx, "Closed stream stderr: %v", stderrR.Close())
			}()

			buf := make([]byte, 4096)
			for {
				_, err := stderrR.Read(buf)
				if err != nil {
					break
				}
			}
		}()

		return &stdoutStreamer{stdoutR, cmd}, nil
	case <-timer:
		defer waitForCommandToFinish()

		// We timed out while waiting for the streaming header to appear,
		// so log an appropriate error message using whatever was printed
		// on stderr
		errMsgFmt := fmt.Sprintf("did not see the %v header after %v seconds:", header, timeout) + " %v"

		stderr, err := ioutil.ReadAll(stderrR)
		if err != nil {
			return nil, fmt.Errorf(
				errMsgFmt,
				fmt.Sprintf("cannot report reason: stderr i/o error: %v", err),
			)
		}

		if len(stderr) == 0 {
			return nil, fmt.Errorf(
				errMsgFmt,
				fmt.Sprintf("cannot report reason: nothing was printed to stderr"),
			)
		}

		return nil, fmt.Errorf(errMsgFmt, string(stderr))
	}
}

// Exec executes a command on the given entry
func (e *externalPluginEntry) Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (ExecResult, error) {
	execResult := ExecResult{}

	// TODO: Figure out how to pass-in opts when we have entries
	// besides Stdin. Could do something like
	//   <plugin_script> exec <path> <state> <opts> <cmd> <args...>
	cmdObj := exec.Command(
		e.script.Path(),
		append(
			[]string{"exec", e.id(), e.state, cmd},
			args...,
		)...,
	)

	// Set-up stdin
	if opts.Stdin != nil {
		cmdObj.Stdin = opts.Stdin
	}

	// Set-up the output streams
	outputCh, stdout, stderr := CreateExecOutputStreams(ctx)
	cmdObj.Stdout = stdout
	cmdObj.Stderr = stderr

	// Start the command
	journal.Record(ctx, "Starting command: %v %v", cmdObj.Path, strings.Join(cmdObj.Args, " "))
	if err := cmdObj.Start(); err != nil {
		stdout.Close()
		stderr.Close()
		return execResult, err
	}

	// Wait for the command to finish
	var exitCode int
	var cmdWaitErr error
	go func() {
		ec, err := ExitCodeFromErr(cmdObj.Wait())
		if err != nil {
			cmdWaitErr = err
		} else {
			exitCode = ec
		}

		stdout.CloseWithError(err)
		stderr.CloseWithError(err)
	}()

	execResult.OutputCh = outputCh
	execResult.ExitCodeCB = func() (int, error) {
		if cmdWaitErr != nil {
			return 0, cmdWaitErr
		}

		return exitCode, nil
	}

	return execResult, nil
}
