package cmd

import (
	"github.com/emirpasic/gods/maps/linkedhashmap"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
	goyaml "gopkg.in/yaml.v2"
)

func infoCommand() *cobra.Command {
	use, aliases := generateShellAlias("info")
	infoCmd := &cobra.Command{
		Use:     use + " <path> [<path>]...",
		Aliases: aliases,
		Short:   "Prints the entries' info at the specified paths",
		Long:    `Print all info Wash has about the specified paths.`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    toRunE(infoMain),
	}
	infoCmd.Flags().StringP("output", "o", "yaml", "Set the output format (json, yaml, or text)")
	return infoCmd
}

func infoMain(cmd *cobra.Command, args []string) exitCode {
	paths := args
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err.Error())
	}

	marshaller, err := cmdutil.NewMarshaller(output)
	if err != nil {
		cmdutil.ErrPrintf(err.Error())
		return exitCode{1}
	}

	conn := cmdutil.NewClient()

	// Fetch the data. We also use a sorted map so that we can control
	// how the information's displayed.
	ec := 0
	infoMap := make(map[string]orderedMap)
	for _, path := range paths {
		entry, err := conn.Info(path)
		if err != nil {
			ec = 1
			cmdutil.ErrPrintf("%v: %v\n", path, err)
			continue
		}

		entryMap := orderedMap{linkedhashmap.New()}
		entryMap.Put("Path", entry.Path)
		entryMap.Put("Name", entry.Name)
		entryMap.Put("CName", entry.CName)
		entryMap.Put("Actions", entry.Actions)
		entryMap.Put("Attributes", entry.Attributes.ToMap(false))

		infoMap[path] = entryMap
	}

	// Marshal the results
	var result interface{} = infoMap
	if len(paths) == 1 {
		// For a single path, it is enough to print the info object
		result = infoMap[paths[0]]
	}
	marshalledResult, err := marshaller.Marshal(result)
	if err != nil {
		cmdutil.ErrPrintf("error marshalling the info results: %v\n", err)
	} else {
		cmdutil.Print(marshalledResult)
	}

	// Return the exit code
	return exitCode{ec}
}

// This wrapped type's here because linkedhashmap doesn't implement the
// json.Marshaler and yaml.Marshaler interfaces.
type orderedMap struct {
	*linkedhashmap.Map
}

func (mp orderedMap) MarshalJSON() ([]byte, error) {
	return mp.ToJSON()
}

// We implement MarshalYAML to preserve each key's ordering.
func (mp orderedMap) MarshalYAML() (interface{}, error) {
	var yamlMap goyaml.MapSlice
	mp.Each(func(key interface{}, value interface{}) {
		yamlMap = append(yamlMap, goyaml.MapItem{
			Key:   key,
			Value: value,
		})
	})
	return yamlMap, nil
}
