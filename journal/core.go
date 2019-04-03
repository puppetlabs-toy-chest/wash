// Package journal provides tools for recording wash operations to journals stored
// in the user's cache directory. The cache directory is created at 'wash/activity'
// in the directory found via https://golang.org/pkg/os/#UserCacheDir. Journals are
// separated by Journal ID.
//
// Wash plugins should use
//	journal.Record(ctx context.Context, msg string, a ...interface{})
// to record entries. The context contains the Journal ID.
package journal

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

// Key is used to identify a Journal ID in a context.
const Key KeyType = iota

var journalCache = datastore.NewMemCacheWithEvicted(closeJournal)
var journalDir = func() string {
	cdir, err := os.UserCacheDir()
	if err != nil {
		panic("Unable to get user cache dir: " + err.Error())
	}
	return filepath.Join(cdir, "wash", "activity")
}()
var expires = 30 * time.Second

// Dir gets the directory where journal entries are written.
func Dir() string {
	return journalDir
}

// SetDir sets the directory where journal entries are written.
func SetDir(dir string) {
	journalDir = dir
}

// GetID returns the Journal ID stored in the context.
func GetID(ctx context.Context) string {
	return ctx.Value(Key).(string)
}

// Record writes a new entry to the journal identified by the ID at `journal.Key` in
// the provided context. It also writes to the server logs at the debug level. If no ID
// is registered, the entry is written to the server logs at the warning level. If the
// ID is an empty string, it uses the ID 'dead-letter-office'.
//
// Record creates a new journal for ID if needed, then appends the message to that journal.
// Records are journaled in the user's cache directory under `wash/activity/ID.log`.
func Record(ctx context.Context, msg string, a ...interface{}) {
	var id string
	if jid, ok := ctx.Value(Key).(string); ok {
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

	obj, err := journalCache.GetOrUpdate(id, expires, true, func() (interface{}, error) {
		jdir := Dir()
		if err := os.MkdirAll(jdir, 0750); err != nil {
			return nil, err
		}

		lpath := filepath.Join(jdir, id+".log")
		f, err := os.OpenFile(lpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return nil, err
		}

		// Use a syncedFile to ensure each write is committed to the disk. The file is only guaranteed to
		// be closed when it's evicted from the cache, which may not happen before shutdown.
		l := &log.Logger{
			Out:       &syncedFile{id: id, file: f},
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

type syncedFile struct {
	id   string
	file *os.File
}

// Write syncs the data immediately after every write operation.
func (sw *syncedFile) Write(b []byte) (n int, err error) {
	n, err = sw.file.Write(b)
	if err := sw.file.Sync(); err != nil {
		log.Warnf("Error syncing journal %v to disk: %v", sw.id, err)
	}
	return n, err
}

func (sw *syncedFile) Close() error {
	return sw.file.Close()
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
