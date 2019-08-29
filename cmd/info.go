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
		Use:     use + " <path>",
		Aliases: aliases,
		Short:   "Prints the entry's info at the specified path",
		Long:    `Print all info Wash has about the specified path.`,
		Args:    cobra.ExactArgs(1),
		RunE:    toRunE(infoMain),
	}
	infoCmd.Flags().StringP("output", "o", "yaml", "Set the output format (json, yaml, or text)")
	infoCmd.Flags().BoolP("include-meta", "", false, "Include the meta attribute")
	return infoCmd
}

func infoMain(cmd *cobra.Command, args []string) exitCode {
	path := args[0]
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		panic(err.Error())
	}
	includeMeta, err := cmd.Flags().GetBool("include-meta")
	if err != nil {
		panic(err.Error())
	}

	marshaller, err := cmdutil.NewMarshaller(output)
	if err != nil {
		cmdutil.ErrPrintf(err.Error())
		return exitCode{1}
	}

	conn := cmdutil.NewClient()

	entry, err := conn.Info(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	// Use a sorted map so that we can control how the information's displayed.
	entryMap := orderedMap{linkedhashmap.New()}
	entryMap.Put("Path", entry.Path)
	entryMap.Put("Name", entry.Name)
	entryMap.Put("CName", entry.CName)
	entryMap.Put("Actions", entry.Actions)
	entryMap.Put("Attributes", entry.Attributes.ToMap(includeMeta))

	marshalledEntry, err := marshaller.Marshal(entryMap)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	cmdutil.Print(marshalledEntry)

	return exitCode{0}
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
