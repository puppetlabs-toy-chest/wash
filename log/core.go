package log

import (
	clog "log"
)

var debug, quiet = false, false

// Init initializes logging format and toggles whether to print debug messages.
func Init(dbg bool, qt bool) {
	debug = dbg
	quiet = qt
	clog.SetFlags(clog.Ldate | clog.Lmicroseconds)
}

// Warnf always prints the message via golang's log package.
func Warnf(format string, v ...interface{}) {
	clog.Printf(format, v...)
}

// Printf prints the message via golang's log package unless Quiet is true.
func Printf(format string, v ...interface{}) {
	if !quiet {
		clog.Printf(format, v...)
	}
}

// Debugf prints the message via golang's log package only if Debug is true.
func Debugf(format string, v ...interface{}) {
	if debug {
		clog.Printf(format, v...)
	}
}
