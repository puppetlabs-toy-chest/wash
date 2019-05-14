package activity

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	log "github.com/sirupsen/logrus"
)

// Journal is used to describe a log of activity. That activity is associated with a user-generated
// event by the ID, and can include a description and start time to provide further context when
// listed in activity history.
type Journal struct {
	ID, Description string
	start           time.Time
}

// NewJournal creates a new journal entry with start time set to 'now'.
func NewJournal(id, desc string) Journal {
	// We set Start based on the time we first encounter the process, not when the process was
	// started. This makes history make a little more sense when interacting with things like
	// the shell, which was likely started before wash was.
	return Journal{ID: id, Description: desc, start: time.Now()}
}

type historyBlob struct {
	mux    sync.Mutex
	list   []Journal
	stored map[string]int
}

var history = initHistory()

func initHistory() historyBlob {
	return historyBlob{
		list:   make([]Journal, 0),
		stored: make(map[string]int),
	}
}

// addToHistory appends the command description to history if it hasn't been registered before.
func (j Journal) addToHistory() {
	// Return if already added to history.
	if _, ok := history.stored[j.ID]; ok {
		return
	}

	// Not yet set. Lock to make sure we only register the command once.
	history.mux.Lock()
	defer history.mux.Unlock()

	// Check again in case something else beat us to it.
	if _, ok := history.stored[j.ID]; ok {
		return
	}

	// Register the command.
	history.stored[j.ID] = len(history.list)
	history.list = append(history.list, j)
}

// Record writes a new entry to the journal. It creates a new file for the journal if needed, then
// appends the message to that journal. Journals are stored in the user's cache directory under
// `wash/activity/ID.log`.
func (j Journal) Record(msg string, a ...interface{}) {
	log.Debugf(msg, a...)

	// This is a single-use cache, so pass in an empty category.
	obj, err := journalFileCache.GetOrUpdate("", j.ID, expires, true, func() (interface{}, error) {
		jpath := j.filepath()
		if err := os.MkdirAll(filepath.Dir(jpath), 0750); err != nil {
			return nil, err
		}

		f, err := os.OpenFile(jpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return nil, err
		}

		l := &log.Logger{
			Out:       f,
			Level:     log.TraceLevel,
			Formatter: &log.TextFormatter{TimestampFormat: time.RFC3339Nano},
		}
		return l, nil
	})
	if err != nil {
		log.Warnf("Error creating journal %v: %v", j.ID, err)
	}

	obj.(*log.Logger).Printf(msg, a...)
}

// Open returns a reader to read the journal.
func (j Journal) Open() (io.ReadCloser, error) {
	return os.Open(j.filepath())
}

var endOfFileLocation = tail.SeekInfo{Offset: -5, Whence: 2}

// Tail streams updates to the journal.
func (j Journal) Tail() (*tail.Tail, error) {
	return tail.TailFile(j.filepath(), tail.Config{
		Follow:    true,
		MustExist: true,
		Location:  &endOfFileLocation,
		Logger:    tail.DiscardingLogger,
	})
}

func (j Journal) filepath() string {
	return filepath.Join(Dir(), j.ID+".log")
}

func (j Journal) String() string {
	return j.ID
}

// Start returns when the activity related to this journal started.
func (j Journal) Start() time.Time {
	return j.start
}

// History returns the entire history.
func History() []Journal {
	// We only ever add to history, so the slice wont be modified by other operations later.
	return history.list
}
