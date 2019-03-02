package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	log "github.com/sirupsen/logrus"
)

type decodedAttributes struct {
	// Atime, Mtime, and Ctime are in Unix time
	Atime int64         `json:"Atime"`
	Mtime int64         `json:"Mtime"`
	Ctime int64         `json:"Ctime"`
	Mode  os.FileMode   `json:"Mode"`
	Size  uint64        `json:"Size"`
	Valid time.Duration `json:"Valid"`
}

func (a decodedAttributes) toAttributes() Attributes {
	return Attributes{
		Atime: time.Unix(a.Atime, 0),
		Mtime: time.Unix(a.Mtime, 0),
		Ctime: time.Unix(a.Ctime, 0),
		Mode:  a.Mode,
		Size:  a.Size,
		Valid: a.Valid,
	}
}

type decodedCacheTTLs struct {
	LS       time.Duration `json:"ls"`
	Open     time.Duration `json:"open"`
	Metadata time.Duration `json:"metadata"`
}

func (c decodedCacheTTLs) toCacheConfig() *CacheConfig {
	config := newCacheConfig()

	if c.LS != 0 {
		config.SetTTLOf(LS, c.LS*time.Second)
	}
	if c.Open != 0 {
		config.SetTTLOf(Open, c.Open*time.Second)
	}
	if c.Metadata != 0 {
		config.SetTTLOf(Metadata, c.Metadata*time.Second)
	}

	return config
}

// decodedExternalPluginEntry describes a decoded serialized entry.
type decodedExternalPluginEntry struct {
	Name             string            `json:"name"`
	SupportedActions []string          `json:"supported_actions"`
	CacheTTLs        decodedCacheTTLs  `json:"cache_ttls"`
	Attributes       decodedAttributes `json:"attributes"`
	State            string            `json:"state"`
}

func (e decodedExternalPluginEntry) toExternalPluginEntry() (*ExternalPluginEntry, error) {
	if e.Name == "" {
		return nil, fmt.Errorf("the entry name must be provided")
	}
	if e.SupportedActions == nil {
		return nil, fmt.Errorf("the entry's supported actions must be provided")
	}

	entry := &ExternalPluginEntry{
		name:             e.Name,
		supportedActions: e.SupportedActions,
		state:            e.State,
		attr:             e.Attributes.toAttributes(),
		cacheConfig:      e.CacheTTLs.toCacheConfig(),
	}
	return entry, nil
}

// ExternalPluginEntry represents an entry from an external plugin. It consists
// of its name, its object (as serialized JSON), and the path to its
// main plugin script.
type ExternalPluginEntry struct {
	script           ExternalPluginScript
	washPath         string
	name             string
	supportedActions []string
	state            string
	cacheConfig      *CacheConfig
	attr             Attributes
}

// Name returns the entry's name
func (e *ExternalPluginEntry) Name() string {
	return e.name
}

// CacheConfig returns the entry's cache config
func (e *ExternalPluginEntry) CacheConfig() *CacheConfig {
	return e.cacheConfig
}

// LS lists the entry's children, if it has any.
func (e *ExternalPluginEntry) LS(context.Context) ([]Entry, error) {
	stdout, err := e.script.InvokeAndWait("ls", e.washPath, e.state)
	if err != nil {
		return nil, err
	}

	var decodedEntries []decodedExternalPluginEntry
	if err := json.Unmarshal(stdout, &decodedEntries); err != nil {
		return nil, err
	}

	entries := make([]Entry, len(decodedEntries))
	for i, decodedExternalPluginEntry := range decodedEntries {
		entry, err := decodedExternalPluginEntry.toExternalPluginEntry()
		if err != nil {
			return nil, err
		}
		entry.script = e.script
		entry.washPath = e.washPath + "/" + entry.Name()

		entries[i] = entry
	}

	return entries, nil
}

// Open returns the entry's content
func (e *ExternalPluginEntry) Open(context.Context) (SizedReader, error) {
	stdout, err := e.script.InvokeAndWait("open", e.washPath, e.state)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(stdout), nil
}

// Metadata displays the resource's metadata
func (e *ExternalPluginEntry) Metadata(context.Context) (MetadataMap, error) {
	stdout, err := e.script.InvokeAndWait("metadata", e.washPath, e.state)
	if err != nil {
		return nil, err
	}

	var metadata MetadataMap
	if err := json.Unmarshal(stdout, &metadata); err != nil {
		return nil, err
	}

	return metadata, nil
}

// Attr returns the entry's filesystem attributes
func (e *ExternalPluginEntry) Attr() Attributes {
	return e.attr
}

// Stream streams the entry's content
func (e *ExternalPluginEntry) Stream(ctx context.Context) (io.Reader, error) {
	cmd, stdoutR, stderrR, err := CreateCommand(
		e.script.Path,
		"stream",
		e.washPath,
		e.state,
	)

	if err != nil {
		return nil, err
	}

	cmdStr := fmt.Sprintf("%v %v %v %v", e.script.Path, "stream", e.washPath, e.state)

	log.Debugf("Starting command: %v", cmdStr)
	if err := cmd.Start(); err != nil {
		LogErr(stdoutR.Close())
		LogErr(stderrR.Close())
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
		log.Debugf("Waiting for command: %v", cmdStr)
		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				log.Debugf("Failed waiting for command: %v", err)
			}
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
				LogErr(stderrR.Close())
			}()

			buf := make([]byte, 4096)
			for {
				_, err := stderrR.Read(buf)
				if err != nil {
					break
				}
			}
		}()

		return stdoutR, nil
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

// TODO: Implement Exec for external plugins
