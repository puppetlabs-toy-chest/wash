package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin/internal"
)

type decodedCacheTTLs struct {
	List     time.Duration `json:"list"`
	Read     time.Duration `json:"read"`
	Metadata time.Duration `json:"metadata"`
}

// decodedExternalPluginEntry describes a decoded serialized entry.
type decodedExternalPluginEntry struct {
	Name          string           `json:"name"`
	Methods       []string         `json:"methods"`
	SlashReplacer string           `json:"slash_replacer"`
	CacheTTLs     decodedCacheTTLs `json:"cache_ttls"`
	Attributes    EntryAttributes  `json:"attributes"`
	State         string           `json:"state"`
}

func (e decodedExternalPluginEntry) toExternalPluginEntry() (*externalPluginEntry, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("the entry name must be provided")
	}
	if e.Methods == nil {
		return nil, fmt.Errorf("the entry's methods must be provided")
	}
	entry := &externalPluginEntry{
		EntryBase: NewEntryBase(),
		methods:   e.Methods,
		state:     e.State,
	}
	entry.SetName(e.Name)
	entry.SetAttributes(e.Attributes)
	entry.setCacheTTLs(e.CacheTTLs)
	if e.SlashReplacer != "" {
		if len([]rune(e.SlashReplacer)) > 1 {
			msg := fmt.Sprintf("e.SlashReplacer: received string %v instead of a character", e.SlashReplacer)
			panic(msg)
		}

		entry.SetSlashReplacer([]rune(e.SlashReplacer)[0])
	}
	return entry, nil
}

// externalPluginEntry represents an external plugin entry
type externalPluginEntry struct {
	EntryBase
	script  externalPluginScript
	methods []string
	state   string
}

func (e *externalPluginEntry) setCacheTTLs(ttls decodedCacheTTLs) {
	if ttls.List != 0 {
		e.SetTTLOf(ListOp, ttls.List*time.Second)
	}
	if ttls.Read != 0 {
		e.SetTTLOf(OpenOp, ttls.Read*time.Second)
	}
	if ttls.Metadata != 0 {
		e.SetTTLOf(MetadataOp, ttls.Metadata*time.Second)
	}
}

// implements returns true if the entry implements the given method,
// false otherwise
func (e *externalPluginEntry) implements(method string) bool {
	for _, m := range e.methods {
		if method == m {
			return true
		}
	}
	return false
}

func (e *externalPluginEntry) ChildSchemas() []*EntrySchema {
	// ChildSchema's meant for core plugins.
	return []*EntrySchema{}
}

func (e *externalPluginEntry) List(ctx context.Context) ([]Entry, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "list", e)
	if err != nil {
		return nil, err
	}
	var decodedEntries []decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedEntries); err != nil {
		return nil, newStdoutDecodeErr(
			ctx,
			"the entries",
			err,
			stdout,
			"[{\"name\":\"entry1\",\"methods\":[\"list\"]},{\"name\":\"entry2\",\"methods\":[\"list\"]}]",
		)
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

func (e *externalPluginEntry) Open(ctx context.Context) (SizedReader, error) {
	stdout, err := e.script.InvokeAndWait(ctx, "read", e)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(stdout), nil
}

func (e *externalPluginEntry) Metadata(ctx context.Context) (JSONObject, error) {
	if !e.implements("metadata") {
		// The entry does not override the "Metadata" method so invoke
		// the default
		return e.EntryBase.Metadata(ctx)
	}
	stdout, err := e.script.InvokeAndWait(ctx, "metadata", e)
	if err != nil {
		return nil, err
	}
	var metadata JSONObject
	if err := json.Unmarshal(stdout, &metadata); err != nil {
		return nil, newStdoutDecodeErr(
			ctx,
			"the metadata",
			err,
			stdout,
			"{\"key1\":\"value1\",\"key2\":\"value2\"}",
		)
	}
	return metadata, nil
}

func (e *externalPluginEntry) Stream(ctx context.Context) (io.ReadCloser, error) {
	cmd := e.script.NewInvocation(ctx, "stream", e)
	stdoutR, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderrR, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	activity.Record(ctx, "Starting %v", cmd)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// "wait" will be used in Stream's error handlers. It will be wrapped
	// in a "defer" call to ensure that cmd.Wait()'s called once we've read
	// all of stdout/stderr. These are the preconditions specified in
	// exec.Cmd#Wait's docs.
	wait := func() {
		if err := cmd.Wait(); err != nil {
			activity.Record(ctx, "Failed waiting for %v to finish: %v", cmd, err)
		}
	}

	// Wait for the header to appear on stdout. This lets us know that
	// the plugin's ready for streaming.
	header := "200"
	headerRdrCh := make(chan error, 1)
	go func() {
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
		headerRdrCh <- nil
	}()
	timeout := 5 * time.Second
	timer := time.After(timeout)
	select {
	case err := <-headerRdrCh:
		if err != nil {
			defer wait()
			// Try to get more context from stderr
			stderr, readErr := ioutil.ReadAll(stderrR)
			if readErr == nil && len(stderr) != 0 {
				err = fmt.Errorf(string(stderr))
			}
			return nil, fmt.Errorf("failed to read the header: %v", err)
		}
		// err == nil, meaning we've received the header. Keep reading from
		// stderr so that the streaming isn't blocked when its buffer is full.
		go func() {
			_, _ = io.Copy(ioutil.Discard, stderrR)
		}()
		return &stdoutStreamer{cmd, stdoutR}, nil
	case <-timer:
		defer wait()
		// We timed out while waiting for the streaming header to appear.
		// Return an appropriate error message using whatever was printed
		// on stderr.
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

func (e *externalPluginEntry) Exec(ctx context.Context, cmd string, args []string, opts ExecOptions) (ExecCommand, error) {
	// Serialize opts to JSON
	optsJSON, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("could not marshal opts %v into JSON: %v", opts, err)
	}
	// Start the command.
	cmdObj := e.script.NewInvocation(ctx, "exec", e, append([]string{string(optsJSON), cmd}, args...)...)
	execCmd := NewExecCommand(ctx)
	cmdObj.SetStdout(execCmd.Stdout())
	cmdObj.SetStderr(execCmd.Stderr())
	if opts.Stdin != nil {
		cmdObj.SetStdin(opts.Stdin)
	}
	activity.Record(ctx, "Starting %v", cmdObj)
	if err := cmdObj.Start(); err != nil {
		return nil, err
	}
	// internal.Command handles context-cancellation cleanup
	// for us, so we don't have to use execCmd.SetStopFunc.

	// Asynchronously wait for the command to finish
	go func() {
		err := cmdObj.Wait()
		execCmd.CloseStreamsWithError(nil)
		exitCode := cmdObj.ProcessState().ExitCode()
		if exitCode < 0 {
			execCmd.SetExitCodeErr(err)
		} else {
			execCmd.SetExitCode(exitCode)
		}
	}()
	return execCmd, nil
}

type stdoutStreamer struct {
	cmd    *internal.Command
	stdout io.ReadCloser
}

func (s *stdoutStreamer) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *stdoutStreamer) Close() error {
	return s.cmd.Wait()
}

func newStdoutDecodeErr(ctx context.Context, decodedThing string, reason error, stdout []byte, example string) error {
	activity.Record(
		ctx,
		"could not decode %v from stdout\nreceived:\n%v\nexpected something like:\n%v",
		decodedThing,
		strings.TrimSpace(string(stdout)),
		example,
	)
	return fmt.Errorf("could not decode %v from stdout: %v", decodedThing, reason)
}
