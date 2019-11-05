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
	hide            bool
}

// NewJournal creates a new journal entry with start time set to 'now'.
func NewJournal(id, desc string) Journal {
	// We set Start based on the time we first encounter the process, not when the process was
	// started. This makes history make a little more sense when interacting with things like
	// the shell, which was likely started before wash was.
	return Journal{ID: id, Description: desc, start: time.Now()}
}

type historyBlob struct {
	// An RWMutex avoids a concurrent map read/write panic.
	// The latter's possible if a Wash subcommand performs
	// concurrent API calls, where one thread could read
	// history.stored while another thread is writing it.
	mux    sync.RWMutex
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
	// Hidden journals aren't added to history.
	if j.hide {
		return
	}

	// Return if already added to history.
	history.mux.RLock()
	if _, ok := history.stored[j.ID]; ok {
		history.mux.RUnlock()
		return
	}
	history.mux.RUnlock()

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

// Callers retrieving the recorder this way should not use
// recorder.logger since it is not guaranteed that
// recorder.logger != nil. Use getRecorder() instead.
//
// The reason recorder() is here is to separate recording method
// invocations from logging activity. Both are things we want
// to store in the recorder; however, the former should never error.
func (j Journal) recorder() recorder {
	// The error's only relevant if we need the logger
	recorder, _ := j.getRecorder()
	return recorder
}

func (j Journal) getLogger() (*log.Logger, error) {
	recorder, err := j.getRecorder()
	return recorder.logger, err
}

func (j Journal) getRecorder() (recorder, error) {
	// This is a single-use cache, so pass in an empty category.
	obj, err := recorderCache.GetOrUpdate("", j.ID, expires, true, func() (interface{}, error) {
		recorder := newRecorder()

		jpath := j.filepath()
		if err := os.MkdirAll(filepath.Dir(jpath), 0750); err != nil {
			return recorder, err
		}

		f, err := os.OpenFile(jpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			return recorder, err
		}

		l := &log.Logger{
			Out:       f,
			Level:     log.TraceLevel,
			Formatter: &log.TextFormatter{TimestampFormat: time.RFC3339Nano},
		}
		recorder.logger = l
		return recorder, nil
	})
	return obj.(recorder), err
}

// Warnf writes a new entry to the journal at WARN level. It also logs to the shell at the same
// level. It creates a new file for the journal if needed, then appends the message to that
// journal. Journals are stored in the user's cache directory under `wash/activity/ID.log`.
func (j Journal) Warnf(msg string, a ...interface{}) {
	log.Warnf(msg, a...)

	if logger, err := j.getLogger(); err != nil {
		log.Warnf("Error creating journal's logger %v: %v", j.ID, err)
	} else {
		logger.Warnf(msg, a...)
	}
}

// Record writes a new entry to the journal. It creates a new file for the journal if needed, then
// appends the message to that journal. Journals are stored in the user's cache directory under
// `wash/activity/ID.log`.
func (j Journal) Record(msg string, a ...interface{}) {
	log.Printf(msg, a...)

	if logger, err := j.getLogger(); err != nil {
		log.Warnf("Error creating journal's logger %v: %v", j.ID, err)
	} else {
		logger.Printf(msg, a...)
	}
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

type entryType = string
type methodInvocations = map[string]bool

type recorder struct {
	logger *log.Logger
	// mI => methodInvocations
	mIMux *sync.RWMutex
	// Recording the method invocations for each entry type minimizes
	// the possibility of a long running command polluting Wash's
	// Google Analytics data. Without it, something like `find s3` would
	// send many s3ObjectPrefix::List invocations for large S3 buckets.
	methodInvocations map[entryType]methodInvocations
}

func newRecorder() recorder {
	return recorder{
		mIMux:             &sync.RWMutex{},
		methodInvocations: make(map[entryType]methodInvocations),
	}
}

// methodInvoked returns true if "method" was invoked by an instance
// of "entryType". methodInvoked is not thread-safe and should be called
// with `r.mIMux.RLock()`.
func (r recorder) methodInvoked(entryType string, method string) bool {
	methodInvocations, ok := r.methodInvocations[entryType]
	if !ok {
		return false
	}
	return methodInvocations[method]
}

// submitMethodInvocation calls submitter and records an invocation of
// "method" by an instance of "entryType" only if the "method" has not
// previously been invoked by an instance of "entryType".
func (r recorder) submitMethodInvocation(entryType string, method string, submitter func()) {
	r.mIMux.RLock()
	if r.methodInvoked(entryType, method) {
		r.mIMux.RUnlock()
		return
	}
	r.mIMux.RUnlock()

	r.mIMux.Lock()
	methodInvocations, ok := r.methodInvocations[entryType]
	if !ok {
		methodInvocations = make(map[string]bool)
		r.methodInvocations[entryType] = methodInvocations
	}
	if methodInvocations[method] {
		r.mIMux.Unlock()
		return
	}

	methodInvocations[method] = true
	r.mIMux.Unlock()

	// We don't need to lock while submitting. If the submission fails we move on instead of retrying
	// so we can mark that we tried before even submitting.
	submitter()
}
