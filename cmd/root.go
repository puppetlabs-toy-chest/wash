package cmd

import (
	"github.com/spf13/cobra"
)

// RootCommand returns the root command
func RootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "wash",
	}

	rootCmd.AddCommand(serverCommand())
	rootCmd.AddCommand(metaCommand())
	rootCmd.AddCommand(lsCommand())

	return rootCmd
}
