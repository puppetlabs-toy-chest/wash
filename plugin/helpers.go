package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Benchkram/errz"
	log "github.com/sirupsen/logrus"
)

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

/*
CName returns the entry's canonical name, which is what Wash uses
to construct the entry's path (see plugin.Path). The entry's cname
is e.Name(), but with all '/' characters replaced by a '#' character.
CNames are necessary because it is possible for entry names to have '/'es
in them, which is illegal in bourne shells and UNIX-y filesystems.

CNames are unique. CName uniqueness is checked in plugin.CachedList.

NOTE: The '#' character was chosen because it is unlikely to appear in
a meaningful entry's name. If, however, there's a good chance that an
entry's name can contain the '#' character, and that two entries can
have the same cname (e.g. 'foo/bar', 'foo#bar'), then you can use
e.SetSlashReplacementChar(<char>) to change the default slash replacement
character from a '#' to <char>.
*/
func CName(e Entry) string {
	// We make the CName a separate function instead of embedding it
	// in the Entry interface because doing so prevents plugin authors
	// from overriding it.
	return strings.Replace(
		e.Name(),
		"/",
		string(e.slashReplacementChar()),
		-1,
	)
}

// Path returns the entry's path rooted at Wash's mountpoint. This is what
// the API consumes. An entry's path is described as
//     /<plugin_name>/<group1_cname>/<group2_cname>/.../<entry_cname>
//
// NOTE: <plugin_name> is really <plugin_cname>. However since <plugin_name>
// can never contain a '/', <plugin_cname> reduces to <plugin_name>.
func Path(e Entry) string {
	return e.id()
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

func parseMode(mode interface{}) (uint64, error) {
	if uintMode, ok := mode.(uint64); ok {
		return uintMode, nil
	}

	if intMode, ok := mode.(int64); ok {
		return uint64(intMode), nil
	}

	if floatMode, ok := mode.(float64); ok {
		if floatMode != float64(uint64(floatMode)) {
			return 0, fmt.Errorf("could not parse mode: the provided mode %v is a decimal number", floatMode)
		}

		return uint64(floatMode), nil
	}

	strMode, ok := mode.(string)
	if !ok {
		return 0, fmt.Errorf("could not parse mode: the provided mode %v is not a uint64, int64, float64, or string", strMode)
	}

	if intMode, err := strconv.ParseUint(strMode, 0, 32); err == nil {
		return intMode, nil
	}

	return 0, fmt.Errorf("could not parse mode: the provided mode %v is not a octal/hex/decimal number", strMode)
}

// ToFileMode converts a given mode into an os.FileMode object.
// The mode can be either an integer or a string representing
// an octal/hex/decimal number.
func ToFileMode(mode interface{}) (os.FileMode, error) {
	intMode, err := parseMode(mode)
	if err != nil {
		return 0, err
	}

	fileMode := os.FileMode(intMode & 0777)
	for bits, mod := range map[uint64]os.FileMode{
		0140000: os.ModeSocket,
		0120000: os.ModeSymlink,
		// Skip file, absence of these implies a regular file.
		0060000: os.ModeDevice,
		0040000: os.ModeDir,
		0020000: os.ModeCharDevice,
		0010000: os.ModeNamedPipe,
		0004000: os.ModeSetuid,
		0002000: os.ModeSetgid,
		0001000: os.ModeSticky,
	} {
		// Ensure exact match of all bits in the mask.
		if intMode&bits == bits {
			fileMode |= mod
		}
	}

	return fileMode, nil
}

// Attr returns the entry's attributes in a form that's consummable by
// FUSE and the API. This may or may not match entry.Attr(). Specifically,
//   * If entry.Attr() sets the size field to SizeUnknown and the entry supports
//   the Read action, then the returned attributes' size will be set to
//   the size of the entry's content as returned by Open.
//
//   * If entry.Attr() does not set the mode, then the returned attributes'
//   mode will be set to "os.ModeDir | 0550" if the entry supports the List
//   action; otherwise, it will be set to 0440. This is needed by FUSE.
//
//   * Otherwise, the returned attributes will match what's returned by
//   entry.Attr().
//
func Attr(ctx context.Context, entry Entry) (Attributes, error) {
	attr, err := entry.Attr(ctx)
	if err != nil {
		return attr, err
	}

	if attr.Size == SizeUnknown && ReadAction.IsSupportedOn(entry) {
		content, openErr := CachedOpen(ctx, entry.(Readable))
		if openErr != nil {
			err = ErrCouldNotDetermineSizeAttr{openErr.Error()}
			attr.Size = 0
		} else {
			size := content.Size()
			if size < 0 {
				return attr, ErrNegativeSizeAttr{size}
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

	return attr, err
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
