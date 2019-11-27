package plugin

import (
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/mattn/go-isatty"
	"golang.org/x/sys/unix"
)

var isInteractive bool = (isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())) &&
	(isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()))

// InitInteractive is used by Wash commands to set option-specific overrides. Only sets
// interactivity to true if it already was and 'init' is also true.
func InitInteractive(init bool) {
	isInteractive = init && isInteractive
}

// IsInteractive returns true if Wash is running as an interactive session. If false, please don't
// prompt for input on stdin.
func IsInteractive() bool {
	return isInteractive
}

func tcGetpgrp(fd int) (pgrp int, err error) {
	return unix.IoctlGetInt(fd, unix.TIOCGPGRP)
}

func tcSetpgrp(fd int, pgrp int) (err error) {
	// Mimic IoctlSetPointerInt, which is not available on macOS.
	v := int32(pgrp)
	return unix.IoctlSetInt(fd, unix.TIOCSPGRP, int(uintptr(unsafe.Pointer(&v))))
}

// Only allow one Prompt call at a time. This prevents multiple plugins loading concurrently
// from messing things up by calling Prompt concurrently.
var promptMux sync.Mutex

// Prompt prints the supplied message, then waits for input on stdin.
func Prompt(msg string) (string, error) {
	if !IsInteractive() {
		return "", fmt.Errorf("not an interactive session")
	}

	promptMux.Lock()
	defer promptMux.Unlock()

	// Even if Wash is running interactively, it will not have control of STDIN while another command
	// is running within the shell environment. If it doesn't have control and tries to read from it,
	// the read will fail. If we have control, read normally. If not, temporarily acquire control for
	// the current process group while we're prompting, then return it afterward so the triggering
	// command can continue.
	inFd := int(os.Stdin.Fd())
	inGrp, err := tcGetpgrp(inFd)
	if err != nil {
		return "", fmt.Errorf("error getting process group controlling stdin: %v", err)
	}
	curGrp := unix.Getpgrp()

	var v string
	if inGrp == curGrp {
		// We control stdin
		fmt.Fprintf(os.Stderr, "%s: ", msg)
		_, err = fmt.Scanln(&v)
	} else {
		// Need to get control, prompt, then return control.
		if err := tcSetpgrp(inFd, curGrp); err != nil {
			return "", fmt.Errorf("error getting control of stdin: %v", err)
		}
		fmt.Fprintf(os.Stderr, "%s: ", msg)
		_, err = fmt.Scanln(&v)
		if err := tcSetpgrp(inFd, inGrp); err != nil {
			// Panic if we can't return control. A messed up environment that they 'kill -9' is worse.
			panic(err.Error())
		}
	}
	// Return the error set by Scanln.
	return v, err
}
