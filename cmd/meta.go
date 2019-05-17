package cmd

import (
	"fmt"

	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <file>",
		Short: "Prints the metadata of a file",
		Args:  cobra.ExactArgs(1),
	}

	metaCmd.Flags().StringP("output", "o", "json", "Set the output format (json or yaml)")
	if err := viper.BindPFlag("output", metaCmd.Flags().Lookup("output")); err != nil {
		cmdutil.ErrPrintf("%v\n", err)
	}

	metaCmd.RunE = toRunE(metaMain)

	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]

	output := viper.GetString("output")
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
