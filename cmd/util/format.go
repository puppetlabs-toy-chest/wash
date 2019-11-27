// Package cmdutil provides utilities for formatting CLI output.
package cmdutil

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Stdout represents Stdout
var Stdout io.Writer = os.Stdout

// Stderr represents Stderr
var Stderr io.Writer = os.Stderr

// ColoredStderr represents a color supporting writer for Stderr
var ColoredStderr io.Writer = color.Error

var stdoutMux sync.Mutex

var stderrMux sync.Mutex

// ErrPrintf formats and prints the provided format string and args on stderr and
// colors the output red.
func ErrPrintf(msg string, a ...interface{}) {
	_, err := fmt.Fprintf(ColoredStderr, color.RedString(msg), a...)
	if err != nil {
		panic(err)
	}
}

// SafeErrPrintf is a thread-safe wrapper to ErrPrintf
func SafeErrPrintf(msg string, a ...interface{}) {
	stderrMux.Lock()
	defer stderrMux.Unlock()
	ErrPrintf(msg, a...)
}

// Printf is a wrapper to fmt.Printf that prints to cmdutil.Stdout
func Printf(msg string, a ...interface{}) {
	_, err := fmt.Fprintf(Stdout, msg, a...)
	if err != nil {
		panic(err)
	}
}

// SafePrintf is a thread-safe wrapper to Printf
func SafePrintf(msg string, a ...interface{}) {
	stdoutMux.Lock()
	defer stdoutMux.Unlock()
	Printf(msg, a...)
}

// Println is a wrapper to fmt.Println that prints to cmdutil.Stdout
func Println(a ...interface{}) {
	_, err := fmt.Fprintln(Stdout, a...)
	if err != nil {
		panic(err)
	}
}

// SafePrintln is a thread-safe wrapper to Println
func SafePrintln(a ...interface{}) {
	stdoutMux.Lock()
	defer stdoutMux.Unlock()
	Println(a...)
}

// Print is a wrapper to fmt.Print that prints to cmdutil.Stdout
func Print(a ...interface{}) {
	_, err := fmt.Fprint(Stdout, a...)
	if err != nil {
		panic(err)
	}
}

// SafePrint is a thread-safe wrapper to Print
func SafePrint(a ...interface{}) {
	stdoutMux.Lock()
	defer stdoutMux.Unlock()
	Print(a...)
}

// FormatDuration formats a duration as `[[dd-]hh:]mm:ss` according to
// http://pubs.opengroup.org/onlinepubs/9699919799/utilities/ps.html.
func FormatDuration(dur time.Duration) string {
	const Decisecond = 100 * time.Millisecond
	const Day = 24 * time.Hour
	d := dur / Day
	dur = dur % Day
	h := dur / time.Hour
	dur = dur % time.Hour
	m := dur / time.Minute
	dur = dur % time.Minute
	s := dur / time.Second
	dur = dur % time.Second
	f := dur / Decisecond
	if d >= 1 {
		return fmt.Sprintf("%02d-%02d:%02d:%02d.%02d", d, h, m, s, f)
	} else if h >= 1 {
		return fmt.Sprintf("%02d:%02d:%02d.%02d", h, m, s, f)
	} else {
		return fmt.Sprintf("%02d:%02d.%02d", m, s, f)
	}
}
