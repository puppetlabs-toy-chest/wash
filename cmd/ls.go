package cmd

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

func lsCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:   "ls [<path>]...",
		Short: "Lists the children of the specified paths, or current directory if not specified",
		Long: `Lists the children of the specified paths, or current directory if
no path is specified. If the -l option is set, then the name,
last modified time, and supported actions are displayed for
each child.`,
		RunE: toRunE(lsMain),
	}
	lsCmd.Flags().BoolP("long", "l", false, "List in long format")
	return lsCmd
}

func formatTime(t time.Time) string {
	return t.Format(time.RFC822)
}

func cname(entry apitypes.Entry) string {
	cname := entry.CName
	if entry.Supports(plugin.ListAction()) {
		cname += "/"
	}
	return cname
}

// item should be a "file"/"dir" type item. formatItem returns
// an array of rows representing that item's entries
func formatItem(item lsItem, longFormat bool) [][]string {
	var entries []apitypes.Entry
	if item.Type() != dirItem {
		// Print the path for "file" items. This is consistent
		// with the built-in ls
		item.entry.CName = item.path
		entries = append(entries, item.entry)
	} else {
		entries = item.children
	}

	var rows [][]string
	for _, entry := range entries {
		var row []string

		if !longFormat {
			row = []string{cname(entry)}
		} else {
			var mtimeStr string
			if entry.Attributes.HasMtime() {
				mtimeStr = formatTime(entry.Attributes.Mtime())
			} else {
				mtimeStr = "<mtime unknown>"
			}
			verbs := strings.Join(entry.Actions, ", ")
			row = []string{cname(entry), mtimeStr, verbs}
		}

		rows = append(rows, row)
	}

	return rows
}

func pad(str string, longFormat bool) []string {
	if longFormat {
		return []string{str, "", ""}
	}
	return []string{str}
}

func lsMain(cmd *cobra.Command, args []string) exitCode {
	paths := []string{"."}
	if len(args) > 0 {
		paths = args
	}
	longFormat, err := cmd.Flags().GetBool("long")
	if err != nil {
		panic(err.Error())
	}

	conn := cmdutil.NewClient()
	items := make([]lsItem, len(paths))

	// Fetch the required data
	var wg sync.WaitGroup
	for ix, path := range paths {
		wg.Add(1)
		go func(ix int, path string) {
			defer wg.Done()

			var item lsItem
			item.path = path
			item.entry, item.err = conn.Info(path)
			if item.err == nil && item.Type() == dirItem {
				item.children, item.err = conn.List(path)
			}

			items[ix] = item
		}(ix, path)
	}
	wg.Wait()

	// Sort the items to ensure that the output's
	// printed in the expected "errors", "files",
	// and "dirs" order.
	sort.Slice(items, func(i int, j int) bool {
		itemOne := items[i]
		itemTwo := items[j]

		return (itemOne.Type() < itemTwo.Type()) ||
			((itemOne.Type() == itemTwo.Type()) && (itemOne.path < itemTwo.path))
	})

	// Partition the items. This makes it easier to print them.
	itemSlice := make(map[int][]lsItem)
	for _, item := range items {
		itemSlice[item.Type()] = append(itemSlice[item.Type()], item)
	}
	errorItems, fileItems, dirItems := itemSlice[errorItem], itemSlice[fileItem], itemSlice[dirItem]

	// Print the items out. Start with the "error" items.
	ec := 0
	for _, item := range errorItems {
		ec = 1
		cmdutil.ErrPrintf("ls: %v: %v\n", item.path, item.err)
	}
	// Now print the "file"/"dir" items as a table to maintain
	// consistent padding. To do that, we'll need to generate
	// the table's rows. Start with the "file" items
	var rows [][]string
	for _, item := range fileItems {
		rows = append(rows, formatItem(item, longFormat)...)
	}
	// Now move on to the "dir" items
	newline := pad("", longFormat)
	if len(items) != len(dirItems) {
		// An "error"/"file" item was printed so include a newline
		rows = append(rows, newline)
	}
	multiplePaths := len(items) > 1
	for ix, item := range dirItems {
		if multiplePaths {
			rows = append(rows, pad(fmt.Sprintf("%v:", item.path), longFormat))
		}
		rows = append(rows, formatItem(item, longFormat)...)
		if ix != (len(dirItems) - 1) {
			rows = append(rows, newline)
		}
	}
	// Now create and print the table using the generated rows
	if len(rows) > 0 {
		cmdutil.Print(cmdutil.NewTable(rows...).Format())
	}

	// Return the exit code
	return exitCode{ec}
}

// There's three possible types of lsItems:
//   * An "error" -- entry that resulted in a failed API request
//   * A "file"   -- entry that does not implement "list"
//   * A "dir"    -- entry that does implement "list"
type lsItem struct {
	path     string
	entry    apitypes.Entry
	children []apitypes.Entry
	err      error
}

const (
	errorItem = 0
	fileItem  = 1
	dirItem   = 2
)

func (item lsItem) Type() int {
	if item.err != nil {
		return errorItem
	}
	if !item.entry.Supports(plugin.ListAction()) {
		return fileItem
	}
	return dirItem
}
