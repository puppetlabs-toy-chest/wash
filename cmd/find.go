package cmd

import (
	"github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/cmd/internal/find"
	"github.com/spf13/cobra"
)

func findCommand() *cobra.Command {
	findCmd := &cobra.Command{
		// `wash find` parses its own flags to keep its syntax consistent with the
		// existing `find` command
		DisableFlagParsing: true,
		Use:                "find",
		Short:              "Prints out all entries that satisfy the given expression",
		RunE:               toRunE(findMain),
	}
	findCmd.SetUsageFunc(func(_ *cobra.Command) error {
		cmdutil.Print(find.Usage())
		return nil
	})
	findCmd.SetHelpTemplate(`{{.UsageString}}`)

	// This tells Cobra to stop parsing CLI flags after the first positional
	// argument. We need it so that Cobra does not interpet our primaries (e.g.
	// like -name) as single-dash flags.
	findCmd.Flags().SetInterspersed(false)

	return findCmd
}

func findMain(cmd *cobra.Command, args []string) exitCode {
	return exitCode{find.Main(args)}
}
