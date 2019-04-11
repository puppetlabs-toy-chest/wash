package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api/client"
	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
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
		{ShortName: "size", FullName: "NAME"},
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

		actions := entry.Actions
		sort.Strings(actions)
		verbs := strings.Join(actions, ", ")

		if entry.Path == apiPath {
			// Represent the pwd as "."
			entry.CName = "."
		}
		name := entry.CName

		isDir := actions[sort.SearchStrings(actions, "list")] == "list"
		if isDir {
			name += "/"
		}

		table[i] = []string{name, ctimeStr, verbs}
	}
	return cmdutil.FormatTable(headers, table)
}

func listMain(cmd *cobra.Command, args []string) exitCode {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}

		path = cwd
	}

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	ls, err := conn.List(apiPath)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	fmt.Print(formatListEntries(apiPath, ls))
	return exitCode{0}
}
