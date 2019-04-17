// Package activity provides tools for recording wash operations to journals stored
// in the user's cache directory. The cache directory is created at 'wash/activity'
// in the directory found via https://golang.org/pkg/os/#UserCacheDir. Journals are
// separated by Journal ID.
//
// Wash plugins should use
//	activity.Record(ctx context.Context, msg string, a ...interface{})
// to record entries. The context contains the Journal ID.
package activity

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/puppetlabs/wash/datastore"
	log "github.com/sirupsen/logrus"
)

// KeyType is used to type keys for looking up context values.
type KeyType int

// JournalKey is used to identify a Journal in a context.
const JournalKey KeyType = iota

var journalFileCache = datastore.NewMemCacheWithEvicted(closeJournal)
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
	journalFileCache.Flush()
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

func closeJournal(id string, obj interface{}) {
	logger, ok := obj.(*log.Logger)
	if !ok {
		// Should always be a logger.
		panic(fmt.Sprintf("journal entry %+v was not a Logger", obj))
	}
	out, ok := logger.Out.(io.Closer)
	if !ok {
		// Should always contain an os.File
		panic(fmt.Sprintf("output entry %+v was not a Closer", logger.Out))
	}
	if err := out.Close(); err != nil {
		log.Warnf("Failed closing journal %v: %v", id, err)
	}
}
