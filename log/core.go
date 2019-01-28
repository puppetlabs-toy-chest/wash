package log

import (
	clog "log"
)

var debug = false

// Init initializes logging format and toggles whether to print debug messages.
func Init(dbg bool) {
	debug = dbg
	clog.SetFlags(clog.Ldate | clog.Lmicroseconds)
}

// Printf always prints the message via golang's log package.
func Printf(format string, v ...interface{}) {
	clog.Printf(format, v...)
}

// Debugf prints the message via golang's log package only if Debug is true.
func Debugf(format string, v ...interface{}) {
	if debug {
		clog.Printf(format, v...)
	}
}
