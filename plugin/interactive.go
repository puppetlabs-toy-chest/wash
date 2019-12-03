package plugin

import (
	"fmt"
	"os"
	"sync"

	"github.com/mattn/go-isatty"
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

	var v string
	fmt.Fprintf(os.Stderr, "%s: ", msg)
	_, err := fmt.Scanln(&v)
	return v, err
}
