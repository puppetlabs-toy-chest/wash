package cmd

import (
	"sync"

	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func metaCommand() *cobra.Command {
	metaCmd := &cobra.Command{
		Use:   "meta <path> [<path>]...",
		Short: "Prints the metadata of the given entries",
		Long: `Prints the metadata of the given entries. By default, meta prints the
full metadata as returned by the metadata endpoint. Specify the
--partial flag to instead print the partial metadata, a (possibly)
reduced set of metadata that's returned when entries are enumerated.`,
		Args: cobra.MinimumNArgs(1),
		RunE: toRunE(metaMain),
	}
	metaCmd.Flags().StringP("output", "o", "yaml", "Set the output format (json, yaml, or text)")
	metaCmd.Flags().BoolP("partial", "p", false, "Print the partial metadata instead")
	return metaCmd
}

func metaMain(cmd *cobra.Command, args []string) exitCode {
	paths := args
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err.Error())
	}
	showPartialMetadata, err := cmd.Flags().GetBool("partial")
	if err != nil {
		panic(err.Error())
	}

	marshaller, err := cmdutil.NewMarshaller(output)
	if err != nil {
		cmdutil.ErrPrintf(err.Error())
		return exitCode{1}
	}

	conn := cmdutil.NewClient()
	metadataMap := make(map[string]map[string]interface{})

	// Fetch the data.
	ec := 0
	var metadataMapMux sync.Mutex
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			var metadata map[string]interface{}

			if showPartialMetadata {
				e, err := conn.Info(path)
				if err != nil {
					ec = 1
					cmdutil.SafeErrPrintf("%v: %v\n", path, err)
					return
				}
				metadata = e.Metadata
			} else {
				var err error
				metadata, err = conn.Metadata(path)
				if err != nil {
					ec = 1
					cmdutil.SafeErrPrintf("%v: %v\n", path, err)
					return
				}
			}

			metadataMapMux.Lock()
			metadataMap[path] = metadata
			metadataMapMux.Unlock()
		}(path)
	}
	wg.Wait()

	// Marshal the results
	var result interface{} = metadataMap
	if len(paths) == 1 {
		// For a single path, it is enough to print the metadata object
		result = metadataMap[paths[0]]
	}
	marshalledResult, err := marshaller.Marshal(result)
	if err != nil {
		cmdutil.ErrPrintf("error marshalling the meta results: %v\n", err)
	} else {
		cmdutil.Print(marshalledResult)
	}

	// Return the exit code
	return exitCode{ec}
}
