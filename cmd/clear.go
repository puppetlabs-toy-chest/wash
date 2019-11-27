package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func clearCommand() *cobra.Command {
	use, aliases := generateShellAlias("clear")
	clearCmd := &cobra.Command{
		Use:     use + " [<path>]...",
		Aliases: aliases,
		Short:   "Clears the cache at the specified paths, or current directory if not specified",
		Long: `Wash caches most operations. If the resource you're querying appears out-of-date, use this
subcommand to reset the cache for resources at or contained within the specified paths.
Defaults to the current directory if no path is provided.`,
		RunE: toRunE(clearMain),
	}
	clearCmd.Flags().BoolP("verbose", "v", false, "Print paths that were cleared from the cache")
	return clearCmd
}

func clearMain(cmd *cobra.Command, args []string) exitCode {
	paths := []string{"."}
	if len(args) > 0 {
		paths = args
	}
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		panic(err.Error())
	}

	conn := cmdutil.NewClient()

	// Perform the operation. Note that wclear isn't parallelized because
	// conn.Clear only hits the Wash daemon. Thus, it should be a very fast
	// request.
	ec := 0
	for _, path := range paths {
		cleared, err := conn.Clear(path)
		if err != nil {
			ec = 1
			cmdutil.ErrPrintf("%v: %v\n", path, err)
			continue
		}

		if verbose {
			for _, p := range cleared {
				cmdutil.Println("Cleared", p)
			}
		} else {
			cmdutil.Println("Cleared", path)
		}
	}

	// Return the exit code
	return exitCode{ec}
}
