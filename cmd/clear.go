package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func clearCommand() *cobra.Command {
	use, aliases := generateShellAlias("clear")
	clearCmd := &cobra.Command{
		Use:     use + " [<path>]",
		Aliases: aliases,
		Short:   "Clears the cache at <path>, or current directory if not specified",
		Long: `Wash caches most operations. If the resource you're querying appears out-of-date, use this
subcommand to reset the cache for resources at or contained within <path>. Defaults to the current
directory if <path> is not specified.`,
		Args: cobra.MaximumNArgs(1),
		RunE: toRunE(clearMain),
	}
	clearCmd.Flags().BoolP("verbose", "v", false, "Print paths that were cleared from the cache")
	return clearCmd
}

func clearMain(cmd *cobra.Command, args []string) exitCode {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		panic(err.Error())
	}

	conn := cmdutil.NewClient()
	cleared, err := conn.Clear(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	if verbose {
		for _, p := range cleared {
			cmdutil.Println("Cleared", p)
		}
	} else {
		cmdutil.Println("Cleared", path)
	}

	return exitCode{0}
}
