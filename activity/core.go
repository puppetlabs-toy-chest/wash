// Package activity provides tools for recording wash operations to journals stored
// in the user's cache directory. The cache directory is created at 'wash/activity'
// in the directory found via https://golang.org/pkg/os/#UserCacheDir. Journals are
// separated by Journal ID.
//
// Wash plugins should use
//	activity.Record(ctx context.Context, msg string, a ...interface{})
// to record entries, and
//  activity.Warnf(ctx context.Context, msg string, a ...interface{})
// to warn about errors. The context contains the Journal ID.
package activity

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/puppetlabs/wash/analytics"
	"github.com/puppetlabs/wash/datastore"
	log "github.com/sirupsen/logrus"
)

// KeyType is used to type keys for looking up context values.
type KeyType int

// JournalKey is used to identify a Journal in a context.
const JournalKey KeyType = iota

// Enforce a limit on cache size to avoid running out of file descriptors. It'll be rare that we
// have dozens of processes running simultaneously.
var recorderCache = datastore.NewMemCache().WithEvicted(closeRecorder).Limit(50)
var journalDir = func() string {
	cdir, err := os.UserCacheDir()
	if err != nil {
		panic("Unable to get user cache dir: " + err.Error())
	}
	return filepath.Join(cdir, "wash", "activity")
}()
var expires = 30 * time.Second

// CloseAll ensures open journals are flushed to disk and closed.
// Use when the application is shutting down.
func CloseAll() {
	recorderCache.Flush()
}

// Dir gets the directory where journals are stored.
func Dir() string {
	return journalDir
}

// SetDir sets the directory where journals are stored.
func SetDir(dir string) {
	journalDir = dir
}

// GetJournal returns the Journal stored in the context.
func GetJournal(ctx context.Context) Journal {
	return ctx.Value(JournalKey).(Journal)
}

var deadLetterOfficeJournal = Journal{ID: "dead-letter-office"}

// Record writes a new entry to the journal identified by the ID at `activity.JournalKey` in
// the provided context. It also writes to the server logs at the debug level. If no ID
// is registered, the entry is written to the server logs at the info level. If the
// ID is an empty string - which can happen when the JournalID header is missing from an
// API call - it uses the ID 'dead-letter-office'.
func Record(ctx context.Context, msg string, a ...interface{}) {
	journal, ok := ctx.Value(JournalKey).(Journal)
	if !ok {
		log.Infof(msg, a...)
		return
	}

	if journal.ID == "" {
		journal = deadLetterOfficeJournal
	} else {
		journal.addToHistory()
	}

	journal.Record(msg, a...)
}

// Warnf writes a new entry to the journal identified by the ID at `activity.JournalKey` in the
// provided context at WARN level. It also writes to the server logs at the same level. Use Warnf
// in plugin `Init` methods for issues during setup that will result in degraded behavior.
func Warnf(ctx context.Context, msg string, a ...interface{}) {
	journal, ok := ctx.Value(JournalKey).(Journal)
	if !ok {
		log.Warnf(msg, a...)
		return
	}

	if journal.ID == "" {
		journal = deadLetterOfficeJournal
	} else {
		journal.addToHistory()
	}

	journal.Warnf(msg, a...)
}

// SubmitMethodInvocation submits a method invocation event to Google Analytics.
// It then records the invocation to the journal identified by the ID at `activity.JournalKey`
// in the provided context.
func SubmitMethodInvocation(ctx context.Context, plugin string, entryType string, method string) {
	// Note that we could move this over to the plugin package. Doing so would shorten
	// the type-signature to (ctx, entry, method), but it would also require us to export
	// the recorder (bad because that is internal to the activity package), or to add some
	// exported methods to the Journal type for atomically reading/writing method invocations
	// (where in-between the reads/writes, we'd also want to execute the analytics submission
	// to ensure everything is atomic). Either approach obfuscates the code for a (small) semantic
	// gain.
	journal, ok := ctx.Value(JournalKey).(Journal)
	if !ok {
		return
	}

	if journal.ID == "" {
		journal = deadLetterOfficeJournal
	} else {
		journal.addToHistory()
	}

	recorder := journal.recorder()

	// Check if the method was already invoked
	recorder.mIMux.RLock()
	if recorder.methodInvoked(entryType, method) {
		recorder.mIMux.RUnlock()
		return
	}
	// Method wasn't invoked
	recorder.mIMux.RUnlock()

	recorder.mIMux.Lock()
	defer recorder.mIMux.Unlock()

	// Check the invocations again in case someone beat us to it
	if recorder.methodInvoked(entryType, method) {
		recorder.mIMux.RUnlock()
		return
	}
	err := analytics.GetClient(ctx).Event(
		"Invocation",
		"Method",
		analytics.Params{
			"Label":      method,
			"Plugin":     plugin,
			"Entry Type": entryType,
		},
	)
	if err != nil {
		// We should never hit this code-path
		panic(fmt.Sprintf("Unexpected error when submitting the method invocation: %v", err))
	}
	recorder.recordMethodInvocation(entryType, method)
}

func closeRecorder(id string, obj interface{}) {
	recorder, ok := obj.(recorder)
	if !ok {
		// Should always be a recorder
		panic(fmt.Sprintf("journal recorder %+v was not a recorder", obj))
	}
	logger := recorder.logger
	out, ok := logger.Out.(io.Closer)
	if !ok {
		// Should always contain an os.File
		panic(fmt.Sprintf("output entry %+v was not a Closer", logger.Out))
	}
	if err := out.Close(); err != nil {
		log.Warnf("Failed closing journal recorder %v: %v", id, err)
	}
}
