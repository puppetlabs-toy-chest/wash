package plugin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

// withConsole invokes the specified function while Wash has control of console input.
// The function should save results by writing to captured variables.
func withConsole(ctx context.Context, fn func(context.Context) error) error {
	if !IsInteractive() {
		// If not interactive, all we can do is call the function and return.
		// If the function prompts for input without checking whether it can, then it may crash.
		return fn(ctx)
	}

	// Even if Wash is running interactively, it will not have control of STDIN while another command
	// is running within the shell environment. If it doesn't have control and tries to read from it,
	// the read will fail. If we have control, read normally. If not, temporarily acquire control for
	// the current process group while we're prompting, then return it afterward so the triggering
	// command can continue.
	inFd := int(os.Stdin.Fd())
	inGrp, err := tcGetpgrp(inFd)
	if err != nil {
		return fmt.Errorf("error getting process group controlling stdin: %v", err)
	}
	curGrp := unix.Getpgrp()

	if inGrp == curGrp {
		return fn(ctx)
	}

	// Catch Ctrl-C while we have input control. Otherwise the shell exits.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)

	// On Ctrl-C, cancel the function call.
	cancelCtx, cancel := context.WithCancel(ctx)
	go func() {
		<-sigCh
		cancel()
	}()

	// Cleanup when the function exits.
	defer func() {
		// Reset the signal watch first so we know there won't be any more attempts to
		// write to sigCh after we close it.
		signal.Reset(syscall.SIGINT)
		close(sigCh)
	}()

	// Need to get control, call the function, then return control.
	if err := tcSetpgrp(inFd, curGrp); err != nil {
		return fmt.Errorf("error getting control of stdin: %v", err)
	}

	// Restore input control when we return.
	defer func() {
		if err := tcSetpgrp(inFd, inGrp); err != nil {
			// Panic if we can't return control. A messed up environment that they 'kill -9' is worse.
			panic(err.Error())
		}
	}()

	return fn(cancelCtx)
}

// Only allow one Prompt call at a time. This prevents multiple plugins loading concurrently
// from messing things up by calling Prompt concurrently.
var promptMux sync.Mutex

// Prompt prints the supplied message, then waits for input on stdin.
func Prompt(msg string) (v string, err error) {
	if IsInteractive() {
		promptMux.Lock()
		defer promptMux.Unlock()

		fmt.Fprintf(os.Stderr, "%s: ", msg)
		_, err = fmt.Scanln(&v)
	} else {
		err = fmt.Errorf("not an interactive session")
	}
	return
}
