package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <file>",
		Short: "Prints the metadata of a file",
		Args:  cobra.ExactArgs(1),
	}

	metaCmd.Flags().StringP("output", "o", "json", "Set the output format (json or yaml)")
	if err := viper.BindPFlag("output", metaCmd.Flags().Lookup("output")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	metaCmd.RunE = toRunE(metaMain)

	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]

	output := viper.GetString("output")
	var marshaller func(interface{}) ([]byte, error)
	switch output {
	case "json":
		marshaller = func(in interface{}) ([]byte, error) { return json.MarshalIndent(in, "", "  ") }
	case "yaml":
		marshaller = yaml.Marshal
	default:
		fmt.Fprintln(os.Stderr, "output must be either 'json' or 'yaml'")
		return exitCode{1}
	}

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	metadata, err := conn.Metadata(apiPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	prettyMetadata, err := marshaller(metadata)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	fmt.Println(string(prettyMetadata))

	return exitCode{0}
}
