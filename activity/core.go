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

// JournalKey is used to identify a Journal ID in a context.
const JournalKey KeyType = iota

var journalCache = datastore.NewMemCacheWithEvicted(closeJournal)
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
	journalCache.Flush()
}

// Dir gets the directory where journals are stored.
func Dir() string {
	return journalDir
}

// SetDir sets the directory where journals are stored.
func SetDir(dir string) {
	journalDir = dir
}

// GetID returns the Journal ID stored in the context.
func GetID(ctx context.Context) string {
	return ctx.Value(JournalKey).(string)
}

// Record writes a new entry to the journal identified by the ID at `activity.JournalKey` in
// the provided context. It also writes to the server logs at the debug level. If no ID
// is registered, the entry is written to the server logs at the warning level. If the
// ID is an empty string, it uses the ID 'dead-letter-office'.
//
// Record creates a new journal for ID if needed, then appends the message to that journal.
// Records are journaled in the user's cache directory under `wash/activity/ID.log`.
func Record(ctx context.Context, msg string, a ...interface{}) {
	var id string
	if jid, ok := ctx.Value(JournalKey).(string); ok {
		if jid == "" {
			id = "dead-letter-office"
		} else {
			id = jid
		}
	} else {
		log.Warnf(msg, a...)
		return
	}

	log.Debugf(msg, a...)

	// This is a single-use cache, so pass in an empty category.
	obj, err := journalCache.GetOrUpdate("", id, expires, true, func() (interface{}, error) {
		jdir := Dir()
		if err := os.MkdirAll(jdir, 0750); err != nil {
			return nil, err
		}

		lpath := filepath.Join(jdir, id+".log")
		f, err := os.OpenFile(lpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return nil, err
		}

		l := &log.Logger{
			Out:       f,
			Level:     log.TraceLevel,
			Formatter: &log.JSONFormatter{TimestampFormat: time.RFC3339Nano},
		}
		return l, nil
	})
	if err != nil {
		log.Warnf("Error creating journal %v: %v", id, err)
	}

	obj.(*log.Logger).Printf(msg, a...)
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
