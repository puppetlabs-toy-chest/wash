package cmd

import (
	"github.com/puppetlabs/wash/cmd/internal/find"
	"github.com/spf13/cobra"
)

func findCommand() *cobra.Command {
	findCmd := &cobra.Command{
		// `wash find` parses its own flags to keep its syntax consistent with the
		// existing `find` command
		DisableFlagParsing: true,
		Use:                "find <path> [expression]",
		// TODO: More detailed usage. Will need to use custom help text in order to
		// properly enumerate all the primaries.
		Short: "Finds stuff",
	}

	// This tells Cobra to stop parsing CLI flags after the first positional
	// argument. We need it so that Cobra does not interpet our primaries (e.g.
	// like -name) as single-dash flags.
	findCmd.Flags().SetInterspersed(false)

	findCmd.RunE = toRunE(findMain)

	return findCmd
}

func findMain(cmd *cobra.Command, args []string) exitCode {
	return exitCode{find.Main(cmd, args)}
}
