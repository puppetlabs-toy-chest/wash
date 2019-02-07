package log

import (
	clog "log"
	"sync"
)

var debug = false

// TODO: why does this help?
var mux = sync.Mutex{}

// Init initializes logging format and toggles whether to print debug messages.
func Init(dbg bool) {
	debug = dbg
	clog.SetFlags(clog.Ldate | clog.Lmicroseconds)
}

// Printf always prints the message via golang's log package.
func Printf(format string, v ...interface{}) {
	mux.Lock()
	clog.Printf(format, v...)
	mux.Unlock()
}

// Debugf prints the message via golang's log package only if Debug is true.
func Debugf(format string, v ...interface{}) {
	if debug {
		mux.Lock()
		clog.Printf(format, v...)
		mux.Unlock()
	}
}
