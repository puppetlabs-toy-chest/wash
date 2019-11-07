package cmd

import (
	"strings"

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

	// Print the description
	description := schema.Description()
	if len(description) > 0 {
		cmdutil.Println(strings.Trim(description, "\n"))
	} else {
		cmdutil.Println("No description provided.")
	}

	// Print the supported signals (if there are any). This part is printed as
	//   SUPPORTED SIGNALS:
	//     * <signal>
	//         <desc>
	//     * <signal>
	//         <desc>
	if len(schema.Signals()) > 0 {
		cmdutil.Println()
		cmdutil.Printf("SUPPORTED SIGNALS\n")
		for signal, description := range schema.Signals() {
			cmdutil.Printf("* %v\n", signal)
			lines := strings.Split(strings.Trim(description, "\n"), "\n")
			for _, line := range lines {
				cmdutil.Printf("    %v\n", line)
			}
		}
	}

	return exitCode{0}
}
