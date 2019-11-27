package cmd

import (
	"sync"

	"github.com/spf13/cobra"

	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

func signalCommand() *cobra.Command {
	signalCmd := &cobra.Command{
		Use:   "signal <signal> [path]...",
		Short: "Sends the specified signal to the entries at the specified paths",
		Args:  cobra.MinimumNArgs(2),
		RunE:  toRunE(signalMain),
	}

	return signalCmd
}

func signalMain(cmd *cobra.Command, args []string) exitCode {
	signal := args[0]
	paths := args[1:]

	conn := cmdutil.NewClient()

	// Perform the operation in parallel
	ec := 0
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			err := conn.Signal(path, signal)
			if err != nil {
				ec = 1
				cmdutil.SafeErrPrintf("%v: %v\n", path, err)
			} else {
				cmdutil.SafePrintf("signalled %v\n", path)
			}
		}(path)
	}
	wg.Wait()

	// Return the exit code
	return exitCode{ec}
}
