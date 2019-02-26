package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/InVisionApp/tabular"
	"github.com/spf13/cobra"

	"github.com/puppetlabs/wash/api"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/config"
)

func lsCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:   "ls [file]",
		Short: "Lists the resources at the indicated path.",
		Args:  cobra.MaximumNArgs(1),
	}

	lsCmd.RunE = toRunE(lsMain)

	return lsCmd
}

func longestFieldFromListing(ls []api.ListEntry, lookup func(api.ListEntry) string) string {
	max := 0
	var match string
	for _, entry := range ls {
		s := lookup(entry)
		l := len(s)
		if l > max {
			max = l
			match = s
		}
	}
	return match
}

func formatTabularListing(ls []api.ListEntry) string {
	var out string

	// Setup the output table
	tab := tabular.New()
	nameWidth := len(longestFieldFromListing(ls, func(e api.ListEntry) string {
		return e.Name
	}))
	verbsWidth := len(longestFieldFromListing(ls, func(e api.ListEntry) string {
		return strings.Join(e.Actions, ", ")
	}))
	tab.Col("size", "NAME", nameWidth+2)
	tab.Col("ctime", "CREATED", 19+2)
	tab.Col("verbs", "ACTIONS", verbsWidth+2)

	table := tab.Parse("*")
	out += fmt.Sprintln(table.Header)

	for _, entry := range ls {
		name := entry.Name

		ctime := entry.Attributes.Ctime

		var ctimeStr string
		if ctime.IsZero() {
			ctimeStr = "<unknown>"
		} else {
			ctimeStr = ctime.Format(time.RFC822)
		}

		actions := entry.Actions
		sort.Strings(actions)
		verbs := strings.Join(actions, ", ")

		isDir := actions[sort.SearchStrings(actions, "list")] == "list"
		if isDir {
			name += "/"
		}

		out += fmt.Sprintf(table.Format, name, ctimeStr, verbs)
	}
	return out
}

func lsMain(cmd *cobra.Command, args []string) exitCode {
	var path string
	if len(args) > 0 {
		path = args[0]
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitCode{1}
		}

		path = cwd
	}

	apiPath, err := client.APIKeyFromPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	conn := client.ForUNIXSocket(config.Socket)

	ls, err := conn.List(apiPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitCode{1}
	}

	// TODO: Handle individual ListEntry errors
	fmt.Print(formatTabularListing(ls))
	return exitCode{0}
}
