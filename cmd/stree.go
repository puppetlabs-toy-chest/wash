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
	for _, path := range paths {
		schema, err := conn.Schema(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		stree := treeprint.New()
		fill(stree, schema)
		stree.SetValue(path)
		cmdutil.Print(stree.String())
	}
	return exitCode{0}
}

func fill(stree treeprint.Tree, schema apitypes.EntrySchema) treeprint.Tree {
	value := schema.Label
	if !schema.Singleton {
		value = fmt.Sprintf("[%v]", value)
	}
	stree.SetValue(value)
	for _, child := range schema.Children {
		// treeprint.Tree has no "AddBranch()" method, so we need to
		// set a stub value. Note that the value will be reset to the
		// correct value in the recursive call, so this is OK.
		subtree := stree.AddBranch("foo")
		fill(subtree, child)
	}
	return stree
}
