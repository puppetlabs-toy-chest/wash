package cmd

import (
	"github.com/spf13/cobra"

	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

func signalCommand() *cobra.Command {
	signalCmd := &cobra.Command{
		Use:   "signal <path> <signal>",
		Short: "Sends the specified signal to the entry at the specified path",
		Args:  cobra.MinimumNArgs(2),
		RunE:  toRunE(signalMain),
	}

	return signalCmd
}

func signalMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]
	signal := args[1]

	conn := cmdutil.NewClient()

	err := conn.Signal(path, signal)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	return exitCode{0}
}
