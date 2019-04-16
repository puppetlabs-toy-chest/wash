package activity

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Journal contains the command-line used to invoke the process associated with this item
// and its journal ID.
type Journal struct {
	ID, Description string
	Start           time.Time
}

var mux = sync.Mutex{}

func initHistory() ([]Journal, map[string]struct{}) {
	return make([]Journal, 0), make(map[string]struct{})
}

// history: a list of commands that have been invoked
// jidToHistory: lookup to identify whether a journal ID has already been recorded in history.
var history, jidToHistory = initHistory()

// registerCommand appends the command description to history if it hasn't been registered before.
func (j Journal) registerCommand() {
	// Return if already added to history.
	if _, ok := jidToHistory[j.ID]; ok {
		return
	}

	// Not yet set. Lock to make sure we only register the command once.
	mux.Lock()
	defer mux.Unlock()

	// Check again in case something else beat us to it.
	if _, ok := jidToHistory[j.ID]; ok {
		return
	}

	// Register the command.
	jidToHistory[j.ID] = struct{}{}
	if j.Start.IsZero() {
		j.Start = time.Now()
	}
	history = append(history, j)
}

// Open returns a reader to read the journal.
func (j Journal) Open() (io.ReadCloser, error) {
	path := filepath.Join(Dir(), j.ID+".log")
	return os.Open(path)
}

func (j Journal) String() string {
	return j.ID
}

// History returns the entire history.
func History() []Journal {
	// We only ever add to history, so the slice wont be modified by other operations later.
	return history
}
