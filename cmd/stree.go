package cmd

import (
	"fmt"

	"github.com/xlab/treeprint"

	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func streeCommand() *cobra.Command {
	streeCmd := &cobra.Command{
		Use:   "stree [<path>...]",
		Short: "Displays the entry's stree (schema-tree)",
		Long: `Displays the entry's stree (schema-tree), which is a high-level overview of the entry's
hierarchy. Non-singleton types are bracketed with "[]".`,
		RunE: toRunE(streeMain),
	}
	return streeCmd
}

func streeMain(cmd *cobra.Command, args []string) exitCode {
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}
	conn := cmdutil.NewClient()
	schemas := make(map[string]*apitypes.EntrySchema)
	for _, path := range paths {
		schema, err := conn.Schema(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		if schema == nil {
			cmdutil.ErrPrintf("%v: schema unknown\n", path)
			continue
		}
		schemas[path] = schema
	}
	for path, schema := range schemas {
		stree := treeprint.New()
		fill(stree, schema, make(map[string]bool))
		stree.SetValue(path)
		cmdutil.Print(stree.String())
	}
	return exitCode{0}
}

func fill(stree treeprint.Tree, schema *apitypes.EntrySchema, visited map[string]bool) treeprint.Tree {
	value := schema.Label()
	if !schema.Singleton() {
		value = fmt.Sprintf("[%v]", value)
	}
	stree.SetValue(value)
	if visited[schema.TypeID()] {
		return stree
	}
	visited[schema.TypeID()] = true
	for _, child := range schema.Children() {
		// treeprint.Tree has no "AddBranch()" method, so we need to
		// set a stub value. Note that the value will be reset to the
		// correct value in the recursive call, so this is OK.
		subtree := stree.AddBranch("foo")
		fill(subtree, child, visited)
	}
	// Delete schema from visited so that stree displays the correct
	// output for siblings that also use schema. Without this code,
	// we wouldn't be able to print the same representation for volume
	// directories in the Kubernetes/Docker/AWS plugins. Instead, only
	// one of those plugins would show the correct representation of
	// "volume_dir => volume_dir, volume_file." The others would only
	// print "volume_dir".
	delete(visited, schema.TypeID())
	return stree
}
