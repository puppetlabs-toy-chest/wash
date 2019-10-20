package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"

	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

func deleteCommand() *cobra.Command {
	deleteCmd := &cobra.Command{
		Use:   "delete <path> [<path>]",
		Short: "Deletes the entries at the specified paths",
		Long: `Deletes the entries at the specified paths, prompting the user for confirmation
before deleting each entry.`,
		Args: cobra.MinimumNArgs(1),
		RunE: toRunE(deleteMain),
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")

	return deleteCmd
}

func deleteMain(cmd *cobra.Command, args []string) exitCode {
	paths := args
	force, err := cmd.Flags().GetBool("force")
	if err != nil {
		panic(err.Error())
	}

	conn := cmdutil.NewClient()

	// Deletion's done in parallel for a better UX.
	var pathsToDelete []string

	// First, confirm the paths to delete.
	for _, path := range paths {
		var deletionConfirmed bool
		if force || !plugin.IsInteractive() {
			deletionConfirmed = true
		} else {
			msg := fmt.Sprintf("remove %v?", path)
			input, err := plugin.Prompt(msg)
			if err != nil {
				cmdutil.ErrPrintf("failed to get confirmation: %v", err)
				return exitCode{1}
			}
			// Assume confirmation if input starts with "y" or "Y". This matches rm.
			deletionConfirmed = len(input) > 0 && (input[0] == 'y' || input[0] == 'Y')
		}
		if deletionConfirmed {
			pathsToDelete = append(pathsToDelete, path)
		}
	}

	// Next, process each request in its own goroutine. We use a writeMux because
	// ErrPrintf is not thread-safe.
	var wg sync.WaitGroup
	var anyFailed bool
	var writeMux sync.Mutex
	for _, path := range pathsToDelete {
		wg.Add(1)
		go func(path string) {
			if err := conn.Delete(path); err != nil {
				writeMux.Lock()
				defer writeMux.Unlock()
				cmdutil.ErrPrintf("%v\n", err)
				anyFailed = true
			}
			wg.Done()
		}(path)
	}
	wg.Wait()
	if anyFailed {
		return exitCode{1}
	}
	return exitCode{0}
}
