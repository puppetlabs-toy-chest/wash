// Package cmdfind stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package cmdfind

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/plugin"
	"github.com/spf13/cobra"
)

// Main is `wash find`'s main function.
func Main(cmd *cobra.Command, args []string) int {
	startTime = time.Now()

	// TODO: Have `wash find` default to recursing on "." (the cwd)
	// if the path is not provided. Also have it handle non-Wash
	// paths.
	path := args[0]
	if path[0] == '-' {
		cmdutil.ErrPrintf("find expects a path")
		return 1
	}
	p, err := parsePredicate(args[1:])
	if err != nil {
		cmdutil.ErrPrintf("find: %v\n", err)
		return 1
	}

	conn := client.ForUNIXSocket(config.Socket)

	e, err := info(&conn, path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return 1
	}
	entries := []entry{e}
	if e.Supports(plugin.ListAction) {
		children, err := list(&conn, e)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return 1
		}
		entries = append(entries, children...)
	}

	for _, e := range entries {
		if p(e) {
			fmt.Printf("%v\n", e.NormalizedPath)
		}
	}
	return 0
}
