package cmd

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdfind "github.com/puppetlabs/wash/cmd/find"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/plugin"
	"github.com/spf13/cobra"
)

func findCommand() *cobra.Command {
	findCmd := &cobra.Command{
		Use: "find <path> [expression]",
		// TODO: More detailed usage. Will need to use custom help text in order to
		// properly enumerate all the primaries.
		Short: "Finds stuff",
		Args:  cobra.MinimumNArgs(1),
	}

	// This tells Cobra to stop parsing CLI flags after the first positional
	// argument. We need it so that Cobra does not interpet our primaries (e.g.
	// like -name) as single-dash flags.
	findCmd.Flags().SetInterspersed(false)

	findCmd.RunE = toRunE(findMain)

	return findCmd
}

func findMain(cmd *cobra.Command, args []string) exitCode {
	cmdfind.SetStartTime(time.Now())

	// TODO: Have `wash find` default to recursing on "." (the cwd)
	// if the path is not provided. Also have it handle non-Wash
	// paths.
	path := args[0]
	if path[0] == '-' {
		cmdutil.ErrPrintf("find expects a path")
		return exitCode{1}
	}
	p, err := cmdfind.ParsePredicate(args[1:])
	if err != nil {
		cmdutil.ErrPrintf("find: %v\n", err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	e, err := conn.Info(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	entries := []apitypes.Entry{e}
	if e.Supports(plugin.ListAction) {
		children, err := conn.List(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		entries = append(entries, children...)
	}

	for _, entry := range entries {
		if p(&entry) {
			// TODO: Include the cwd's directory in path (so that find
			// prints out absolute paths).
			fmt.Printf("%v\n", entry.Path)
		}
	}
	return exitCode{0}
}
