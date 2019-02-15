package cmd

import (
	"bytes"
	"encoding/json"
	"log"
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
	socket := config.Fields.Socket

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		log.Print(err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(socket)

	metadata, err := conn.Metadata(apiPath)
	if err != nil {
		log.Print(err)
		return exitCode{1}
	}

	var prettyMetadata bytes.Buffer
	json.Indent(&prettyMetadata, metadata, "", "  ")

	prettyMetadata.WriteTo(os.Stdout)

	return exitCode{0}
}
