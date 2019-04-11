package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

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

func formatListEntries(apiPath string, ls []apitypes.ListEntry) string {
	headers := []cmdutil.ColumnHeader{
		{ShortName: "name", FullName: "NAME"},
		{ShortName: "ctime", FullName: "CREATED"},
		{ShortName: "verbs", FullName: "ACTIONS"},
	}
	table := make([][]string, len(ls))
	for i, entry := range ls {
		var ctimeStr string
		if entry.Attributes.HasCtime() {
			ctimeStr = entry.Attributes.Ctime().Format(time.RFC822)
		} else {
			ctimeStr = "<unknown>"
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

		table[i] = []string{name, ctimeStr, verbs}
	}
	return cmdutil.FormatTable(headers, table)
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
	return errors.New("Listing non-resource types not yet supported")
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
