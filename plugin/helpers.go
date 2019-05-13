package plugin

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

/*
Name returns the entry's name as it was passed into
plugin.NewEntry. It is meant to be called by other
Wash packages. Plugin authors should use EntryBase#Name
when writing their plugins.
*/
func Name(e Entry) string {
	// The reason we don't expose EntryBase#Name in the Entry
	// interface is so plugin authors don't override it. It ensures
	// that whatever name they pass into plugin.NewEntry is the
	// name received by Wash.
	return e.name()
}

/*
CName returns the entry's canonical name, which is what Wash uses to
construct the entry's path. The entry's cname is plugin.Name(e), but with
all '/' characters replaced by a '#' character. CNames are necessary
because it is possible for entry names to have '/'es in them, which is
illegal in bourne shells and UNIX-y filesystems.

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
		e.name(),
		"/",
		string(e.slashReplacementChar()),
		-1,
	)
}

// ID returns the entry's ID, which is just its path rooted at Wash's mountpoint.
// An entry's ID is described as
//     /<plugin_name>/<group1_cname>/<group2_cname>/.../<entry_cname>
//
// NOTE: <plugin_name> is really <plugin_cname>. However since <plugin_name>
// can never contain a '/', <plugin_cname> reduces to <plugin_name>.
func ID(e Entry) string {
	if e.id() == "" {
		msg := fmt.Sprintf("plugin.ID: entry %v (cname %v) has no ID", e.name(), CName(e))
		panic(msg)
	}
	return e.id()
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

// Attributes returns the entry's attribtues
func Attributes(e Entry) EntryAttributes {
	return e.attributes()
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

	return exitErr.ExitCode(), nil
}
