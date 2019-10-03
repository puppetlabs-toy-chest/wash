package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func docsCommand() *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs <path>",
		Short: "Displays the entry's documentation. This is currently its description.",
		RunE:  toRunE(docsMain),
	}
	return docsCmd
}

func docsMain(cmd *cobra.Command, args []string) exitCode {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	conn := cmdutil.NewClient()

	schema, err := conn.Schema(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}
	if schema == nil {
		cmdutil.ErrPrintf("%v: schema unknown\n", path)
		return exitCode{0}
	}
	description := schema.Description()
	if len(description) > 0 {
		cmdutil.Println(description)
	} else {
		cmdutil.Println("No description provided.")
	}
	return exitCode{0}
}
