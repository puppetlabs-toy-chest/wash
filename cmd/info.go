package cmd

import (
	"fmt"

	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func infoCommand() *cobra.Command {
	infoCmd := &cobra.Command{
		Use:   "info <path>",
		Short: "Prints the entry's info at the specified path",
		Long:  `Print all info Wash has about the specified path, including filesystem attributes and metadata.`,
		Args:  cobra.ExactArgs(1),
		RunE:  toRunE(infoMain),
	}
	infoCmd.Flags().StringP("output", "o", "json", "Set the output format (json or yaml)")
	return infoCmd
}

func infoMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err.Error())
	}

	marshaller, err := cmdutil.NewMarshaller(output)
	if err != nil {
		cmdutil.ErrPrintf(err.Error())
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	entry, err := conn.Info(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	marshalledEntry, err := marshaller.Marshal(entry)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	fmt.Println(marshalledEntry)

	return exitCode{0}
}
