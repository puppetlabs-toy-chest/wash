package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Benchkram/errz"
	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/plugin"
)

func listCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:     "list [file]",
		Aliases: []string{"ls"},
		Short:   "Lists the resources at the indicated path.",
		Args:    cobra.MaximumNArgs(1),
	}

	listCmd.RunE = toRunE(listMain)

	return listCmd
}

func headers() []cmdutil.ColumnHeader {
	return []cmdutil.ColumnHeader{
		{ShortName: "name", FullName: "NAME"},
		{ShortName: "mtime", FullName: "MODIFIED"},
		{ShortName: "verbs", FullName: "ACTIONS"},
	}
}

func format(t time.Time) string {
	return t.Format(time.RFC822)
}

func formatListEntries(apiPath string, ls []apitypes.ListEntry) string {
	table := make([][]string, len(ls))
	for i, entry := range ls {
		var mtimeStr string
		if entry.Attributes.HasMtime() {
			mtimeStr = format(entry.Attributes.Mtime())
		} else {
			mtimeStr = "<unknown>"
		}

		verbs := strings.Join(entry.Actions, ", ")

		name := entry.CName
		if len(ls) > 1 && entry.Path == apiPath {
			// Represent the pwd as "."
			name = "."
		}

		if isListable(entry) {
			name += "/"
		}

		table[i] = []string{name, mtimeStr, verbs}
	}
	return cmdutil.FormatTable(headers(), table)
}

func findEntry(entries []apitypes.ListEntry, name string) apitypes.ListEntry {
	for _, entry := range entries {
		if entry.CName == name {
			return entry
		}
	}
	return apitypes.ListEntry{}
}

func isListable(entry apitypes.ListEntry) bool {
	for _, action := range entry.Actions {
		if action == plugin.ListAction.Name {
			return true
		}
	}
	return false
}

func listResource(apiPath string) error {
	conn := client.ForUNIXSocket(config.Socket)

	var entries []apitypes.ListEntry
	if apiPath == "/" {
		// The root, definitely listable
		ls, err := conn.List(apiPath)
		if err != nil {
			return err
		}
		entries = ls
	} else {
		// List the parent to see whether it's a single entry or a listable resource
		parent, base := filepath.Split(apiPath)
		parentEntries, err := conn.List(parent)
		if err != nil {
			return err
		}

		target := findEntry(parentEntries, base)
		if target.CName != base || isListable(target) {
			// If we didn't find a parent entry, just try listing it. Can happen if the type has changed
			// or disappeared, and List will give a reasonable error in that case.
			ls, err := conn.List(apiPath)
			if err != nil {
				return err
			}
			entries = ls
		} else {
			entries = []apitypes.ListEntry{target}
		}
	}

	fmt.Print(formatListEntries(apiPath, entries))
	return nil
}

func listPath(path string) error {
	finfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	var table [][]string
	if finfo.IsDir() {
		matches, err := filepath.Glob(filepath.Join(path, "*"))
		errz.Fatal(err)
		table = make([][]string, len(matches)+1)
		table[0] = []string{".", format(finfo.ModTime()), "list"}
		for i, match := range matches {
			finfo, err := os.Stat(match)
			if err != nil {
				return err
			}
			actions := "read"
			if finfo.IsDir() {
				actions = "list"
			}
			table[i+1] = []string{finfo.Name(), format(finfo.ModTime()), actions}
		}
	} else {
		table = [][]string{[]string{finfo.Name(), format(finfo.ModTime()), "read"}}
	}
	// Most operating systems don't track when a thing was created, just the last time it was
	// modified. Some filesystems track the inode birth time, but Go doesn't expose that. List
	// modification time for now, and note that this differs from what `ls -l` shows on macOS.
	fmt.Print(cmdutil.FormatTable(headers(), table))
	return nil
}

func listMain(cmd *cobra.Command, args []string) exitCode {
	// If no path is declared, try to list the current directory/resource
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	apiPath, err := client.APIKeyFromPath(path)
	if err == nil {
		err = listResource(apiPath)
	} else {
		err = listPath(path)
	}

	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	return exitCode{0}
}
