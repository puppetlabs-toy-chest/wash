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

var std = datastore.NewMemCacheWithEvicted(closeFile)
var journaldir = func() string {
	cdir, err := os.UserCacheDir()
	if err != nil {
		panic("Unable to get user cache dir: " + err.Error())
	}
	return filepath.Join(cdir, "wash", "activity")
}()
var expires = 30 * time.Second

// Dir gets the directory where journal entries are written.
func Dir() string {
	return journaldir
}

// SetDir sets the directory where journal entries are written.
func SetDir(dir string) {
	journaldir = dir
}

// Record creates a new journal for 'id' if needed, then appends the message to that journal.
// Records are journaled in the user's cache directory under `wash/activity/ID.log`.
func Record(id string, msg string, a ...interface{}) {
	obj, err := std.GetOrUpdate(id, expires, true, func() (interface{}, error) {
		jdir := Dir()
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
