package cmd

import (
	"fmt"

	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <path>",
		Short: "Prints the metadata of a resource",
		Args:  cobra.ExactArgs(1),
		RunE:  toRunE(metaMain),
	}
	metaCmd.Flags().StringP("output", "o", "json", "Set the output format (json or yaml)")
	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
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

	metadata, err := conn.Metadata(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	prettyMetadata, err := marshaller.Marshal(metadata)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	fmt.Println(prettyMetadata)

	return exitCode{0}
}
