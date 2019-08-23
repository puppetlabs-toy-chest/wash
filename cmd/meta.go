package cmd

import (
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <path>",
		Short: "Prints the entry's metadata",
		Long: `Prints the entry's metadata. By default, meta prints the full metadata as returned by the
metadata endpoint. Specify the --attribute flag to instead print the meta attribute, a
(possibly) reduced set of metadata that's returned when entries are enumerated.`,
		Args: cobra.ExactArgs(1),
		RunE: toRunE(metaMain),
	}
	metaCmd.Flags().StringP("output", "o", "yaml", "Set the output format (json or yaml)")
	metaCmd.Flags().BoolP("attribute", "a", false, "Print the meta attribute instead of the full metadata")
	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err.Error())
	}
	showMetaAttr, err := cmd.Flags().GetBool("attribute")
	if err != nil {
		panic(err.Error())
	}

	marshaller, err := cmdutil.NewMarshaller(output)
	if err != nil {
		cmdutil.ErrPrintf(err.Error())
		return exitCode{1}
	}

	conn := cmdutil.NewClient()

	var metadata map[string]interface{}
	if showMetaAttr {
		e, err := conn.Info(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		metadata = e.Attributes.Meta()
	} else {
		metadata, err = conn.Metadata(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
	}

	prettyMetadata, err := marshaller.Marshal(metadata)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	cmdutil.Println(prettyMetadata)

	return exitCode{0}
}
