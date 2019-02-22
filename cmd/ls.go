package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/InVisionApp/tabular"
	"github.com/spf13/cobra"

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

func longestFieldFromListing(ls []client.LSItem, lookup func(client.LSItem) string) string {
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

func formatTabularListing(ls []client.LSItem) string {
	var out string

	// Setup the output table
	tab := tabular.New()
	nameWidth := len(longestFieldFromListing(ls, func(e client.LSItem) string {
		return e.Name
	}))
	verbsWidth := len(longestFieldFromListing(ls, func(e client.LSItem) string {
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
		if ctime == "" {
			ctime = "<unknown>"
		} else {
			t, err := time.Parse("2006-01-02T15:04:05-07:00", ctime)
			if err != nil {
				ctime = fmt.Sprintf("<raw:%s>", err)
			} else {
				ctime = t.Format(time.RFC822)
			}
		}

		actions := entry.Actions
		sort.Strings(actions)
		verbs := strings.Join(actions, ", ")

		isDir := actions[sort.SearchStrings(actions, "list")] == "list"
		if isDir {
			name += "/"
		}

		out += fmt.Sprintf(table.Format, name, ctime, verbs)
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

	fmt.Print(formatTabularListing(ls))
	return exitCode{0}
}
