package cmd

import (
	"encoding/json"
	"os"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <file>",
		Short: "Prints the metadata of a file",
		Args:  cobra.MinimumNArgs(1),
	}

	metaCmd.RunE = toRunE(metaMain)

	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		writeError(err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	metadata, err := conn.Metadata(apiPath)
	if err != nil {
		writeError(err)
		return exitCode{1}
	}

	prettyMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		writeError(err)
		return exitCode{1}
	}
	prettyMetadata = append(prettyMetadata, byte('\n'))

	os.Stdout.Write(prettyMetadata)

	return exitCode{0}
}
