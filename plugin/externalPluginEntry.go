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

	"github.com/puppetlabs/wash/activity"
)

type decodedCacheTTLs struct {
	List     time.Duration `json:"list"`
	Open     time.Duration `json:"open"`
	Metadata time.Duration `json:"metadata"`
}

// decodedExternalPluginEntry describes a decoded serialized entry.
type decodedExternalPluginEntry struct {
	Name                 string           `json:"name"`
	SupportedActions     []string         `json:"supported_actions"`
	SlashReplacementChar string           `json:"slash_replacement_char"`
	CacheTTLs            decodedCacheTTLs `json:"cache_ttls"`
	Attributes           EntryAttributes  `json:"attributes"`
	State                string           `json:"state"`
}

func (e decodedExternalPluginEntry) toExternalPluginEntry() (*externalPluginEntry, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("the entry name must be provided")
	}
	if e.SupportedActions == nil {
		return nil, fmt.Errorf("the entry's supported actions must be provided")
	}

	entry := &externalPluginEntry{
		EntryBase:        NewEntry(e.Name),
		supportedActions: e.SupportedActions,
		state:            e.State,
	}
	entry.SetAttributes(e.Attributes)
	entry.setCacheTTLs(e.CacheTTLs)
	if e.SlashReplacementChar != "" {
		if len([]rune(e.SlashReplacementChar)) > 1 {
			msg := fmt.Sprintf("e.SlashReplacementChar: received string %v instead of a character", e.SlashReplacementChar)
			panic(msg)
		}

		entry.SetSlashReplacementChar([]rune(e.SlashReplacementChar)[0])
	}

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
}

func (e *externalPluginEntry) setCacheTTLs(ttls decodedCacheTTLs) {
	if ttls.List != 0 {
		e.SetTTLOf(ListOp, ttls.List*time.Second)
	}
	if ttls.Open != 0 {
		e.SetTTLOf(OpenOp, ttls.Open*time.Second)
	}
	if ttls.Metadata != 0 {
		e.SetTTLOf(MetadataOp, ttls.Metadata*time.Second)
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
		activity.Record(
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

// Metadata displays the entry's metadata
func (e *externalPluginEntry) Metadata(ctx context.Context) (EntryMetadata, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "metadata", e.id(), e.state)
	if err != nil {
		return nil, err
	}

	var metadata EntryMetadata
	if err := json.Unmarshal(stdout, &metadata); err != nil {
		activity.Record(
			ctx,
			"could not decode the metadata from stdout\nreceived:\n%v\nexpected something like:\n%v",
			strings.TrimSpace(string(stdout)),
			"{\"key1\":\"value1\",\"key2\":\"value2\"}",
		)

		return nil, fmt.Errorf("could not decode the metadata from stdout: %v", err)
	}

	return metadata, nil
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
func (e *externalPluginEntry) Stream(ctx context.Context) (io.ReadCloser, error) {
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

	activity.Record(ctx, "Starting command: %v", cmdStr)
	if err := cmd.Start(); err != nil {
		activity.Record(ctx, "Closed command stdout: %v", stdoutR.Close())
		activity.Record(ctx, "Closed command stderr: %v", stderrR.Close())
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
		activity.Record(ctx, "Waiting for command: %v", cmdStr)
		_, err := ExitCodeFromErr(cmd.Wait())
		if err != nil {
			activity.Record(ctx, "Failed waiting for command: %v", err)
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
				activity.Record(ctx, "Closed stream stderr: %v", stderrR.Close())
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
func (e *externalPluginEntry) Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (*RunningCommand, error) {
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

	runningCmd := NewRunningCommand(ctx)

	// Set-up the output streams
	cmdObj.Stdout = runningCmd.Stdout()
	cmdObj.Stderr = runningCmd.Stderr()

	// Start the command
	activity.Record(ctx, "Starting command: %v %v", cmdObj.Path, strings.Join(cmdObj.Args, " "))
	if err := cmdObj.Start(); err != nil {
		return nil, err
	}

	// Wait for the command to finish
	go func() {
		ec, err := ExitCodeFromErr(cmdObj.Wait())
		if err != nil {
			runningCmd.CloseStreamsWithError(err)
			return
		}
		runningCmd.SetExitCode(ec)
	}()
	
	return runningCmd, nil
}
