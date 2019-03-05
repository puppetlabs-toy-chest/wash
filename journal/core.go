package journal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/puppetlabs/wash/datastore"
	log "github.com/sirupsen/logrus"
)

// TODO: Would perform better if we retrieved from cache once when creating a NamedJournal and
// kept that one active until the NamedJournal is no longer in use.
var std = datastore.NewMemCacheWithEvicted(closeFile)
var cachedir = func() string {
	cdir, err := os.UserCacheDir()
	if err != nil {
		panic("Unable to get user cache dir: " + err.Error())
	}
	return cdir
}()
var expires = 30 * time.Second

func journaldir() string {
	return filepath.Join(cachedir, "wash", "journal")
}

// Log creates a new journal for 'id' if needed, then appends the message to that journal.
// Logs are journaled in the user's cache directory under `wash/journal/ID.log`.
func Log(id string, msg string, a ...interface{}) {
	obj, err := std.GetOrUpdate(id, expires, true, func() (interface{}, error) {
		jdir := journaldir()
		if err := os.MkdirAll(jdir, 0750); err != nil {
			return nil, err
		}

		lpath := filepath.Join(jdir, id+".log")
		f, err := os.OpenFile(lpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return nil, err
		}

		// Use a syncWriter to ensure each write is committed to the disk. The file is only guaranteed to
		// be closed when it's evicted from the cache, which may not happen before shutdown.
		l := log.New()
		l.Out = &syncWriter{id: id, file: f}
		l.Level = log.TraceLevel
		return l, nil
	})
	if err != nil {
		log.Warnf("Error creating journal %v: %v", id, err)
	}

	obj.(*log.Logger).Printf(msg, a...)
}

type syncWriter struct {
	id   string
	file *os.File
}

// Write syncs the data immediately after every write operation.
func (sw *syncWriter) Write(b []byte) (n int, err error) {
	n, err = sw.file.Write(b)
	if err := sw.file.Sync(); err != nil {
		log.Warnf("Error syncing journal %v to disk: %v", sw.id, err)
	}
	return n, err
}

func (sw *syncWriter) Close() error {
	return sw.file.Close()
}

func closeFile(id string, obj interface{}) {
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
