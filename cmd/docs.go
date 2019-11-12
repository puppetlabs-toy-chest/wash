package cmd

import (
	"strings"

	apitypes "github.com/puppetlabs/wash/api/types"
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

	// Print the supported signals/signal groups (if there are any). This part is
	// printed as
	//   SUPPORTED SIGNALS:
	//     * <signal>
	//         <desc>
	//     * <signal>
	//         <desc>
	//
	//   SUPPORTED SIGNAL GROUPS:
	//     * <signal_group>
	//         <desc>
	//     * <signal_group>
	//         <desc>
	if len(schema.Signals()) > 0 {
		var supportedSignals []apitypes.SignalSchema
		var supportedSignalGroups []apitypes.SignalSchema
		for _, signalSchema := range schema.Signals() {
			if signalSchema.IsGroup() {
				supportedSignalGroups = append(supportedSignalGroups, signalSchema)
			} else {
				supportedSignals = append(supportedSignals, signalSchema)
			}
		}
		if len(supportedSignals) > 0 {
			printSignalSet("SUPPORTED SIGNALS", supportedSignals)
		}
		if len(supportedSignalGroups) > 0 {
			printSignalSet("SUPPORTED SIGNAL GROUPS", supportedSignalGroups)
		}
	}

	return exitCode{0}
}

func printSignalSet(setName string, signals []apitypes.SignalSchema) {
	cmdutil.Println()
	cmdutil.Printf("%v\n", setName)
	for _, signal := range signals {
		cmdutil.Printf("* %v\n", signal.Name())
		lines := strings.Split(strings.Trim(signal.Description(), "\n"), "\n")
		for _, line := range lines {
			cmdutil.Printf("    %v\n", line)
		}
	}
}
