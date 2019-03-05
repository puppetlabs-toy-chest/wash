package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/Benchkram/errz"
	log "github.com/sirupsen/logrus"
)

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

// NewEntry creates a new named entry
func NewEntry(name string) EntryBase {
	return EntryBase{name, newCacheConfig()}
}

// ToMetadata converts an object to a metadata result. If the input is already an array of bytes, it
// must contain a serialized JSON object. Will panic if given something besides a struct or []byte.
func ToMetadata(obj interface{}) MetadataMap {
	var err error
	var inrec []byte
	if arr, ok := obj.([]byte); ok {
		inrec = arr
	} else {
		if inrec, err = json.Marshal(obj); err != nil {
			// Internal error if we can't marshal an object
			panic(err)
		}
	}
	var meta MetadataMap
	// Internal error if not a JSON object
	errz.Fatal(json.Unmarshal(inrec, &meta))
	return meta
}

// TrackTime helper is useful for timing functions.
// Use with `defer plugin.TrackTime(time.Now(), "funcname")`.
func TrackTime(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Infof("%s took %s", name, elapsed)
}

// ErrNegativeSizeAttr indicates that a negative value for the
// size attribute was returned
type ErrNegativeSizeAttr struct {
	size int64
}

func (e ErrNegativeSizeAttr) Error() string {
	return fmt.Sprintf("returned a negative value for the size: %v", e.size)
}

// ErrCouldNotDetermineSizeAttr indicates that the size attribute
// could not be determined
type ErrCouldNotDetermineSizeAttr struct {
	reason string
}

func (e ErrCouldNotDetermineSizeAttr) Error() string {
	return fmt.Sprintf("could not determine the size attribute: %v", e.reason)
}

// FillAttr fills the given attributes struct with the entry's attributes.
func FillAttr(ctx context.Context, entry Entry, entryID string, attr *Attributes) error {
	attr.Size = SizeUnknown

	if item, ok := entry.(File); ok {
		attrRet := item.Attr()

		// We need the zero-value check for external plugin entries,
		// b/c the ExternalPluginEntry class has Attr() implemented,
		// but not all external plugin entries have attributes.
		if attrRet != (Attributes{}) {
			(*attr) = attrRet
		}
	}

	var err error
	if ReadAction.IsSupportedOn(entry) && attr.Size == SizeUnknown {
		content, openErr := CachedOpen(ctx, entry.(Readable), entryID)
		if openErr != nil {
			err = ErrCouldNotDetermineSizeAttr{openErr.Error()}
			attr.Size = 0
		} else {
			size := content.Size()
			if size < 0 {
				return ErrNegativeSizeAttr{size}
			}

			attr.Size = uint64(size)
		}
	}

	if attr.Mode == 0 {
		if ListAction.IsSupportedOn(entry) {
			attr.Mode = os.ModeDir | 0550
		} else {
			attr.Mode = 0440
		}
	}

	return err
}

// CreateCommand creates a cmd object encapsulating the given cmd and its args.
// It returns the cmd object + its stdout and stderr pipes.
//
// TODO: Maybe useful to create our own Command object that wraps *exec.Cmd.
// This way, we can extend it. For example, we could add a method that returns the
// full command string, which would be useful for logging.
func CreateCommand(cmd string, args ...string) (*exec.Cmd, io.ReadCloser, io.ReadCloser, error) {
	cmdObj := exec.Command(cmd, args...)

	stdout, err := cmdObj.StdoutPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := cmdObj.StderrPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	return cmdObj, stdout, stderr, nil
}

// ExitCodeFromErr attempts to get the exit-code from the passed-in
// error object. If successful, it returns the exit-code. Otherwise,
// it returns the passed-in error object as the error.
func ExitCodeFromErr(err error) (int, error) {
	if err == nil {
		return 0, nil
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return 0, err
	}

	// For some reason, exitErr.ExitCode() results in a "no field or method"
	// compiler error on some machines. Other variants like
	// exitErr.ProcessState.ExitCode() also don't work. Thus, we use the method
	// described in https://stackoverflow.com/questions/10385551/get-exit-code-go
	// to get the exit code.
	ws := exitErr.Sys().(syscall.WaitStatus)
	return ws.ExitStatus(), nil
}
