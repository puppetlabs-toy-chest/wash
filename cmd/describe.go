package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func describeCommand() *cobra.Command {
	describeCmd := &cobra.Command{
		Use:   "describe <path>",
		Short: "Displays the entry's description (if it has one).",
		Long: `Displays the entry's description (if it has one). An entry will have a description
if what it is is not obvious from its path, or if there are any subtleties involved when invoking
one of its supported actions (like e.g. additional configuration). If the entry's a plugin root,
then the entry's description is the plugin's documentation.`,
		RunE: toRunE(describeMain),
	}
	return describeCmd
}

func describeMain(cmd *cobra.Command, args []string) exitCode {
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
	}
	return exitCode{0}
}
