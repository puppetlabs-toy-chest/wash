package cmd

import (
	"strings"
	"time"

	"github.com/spf13/cobra"

	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

func lsCommand() *cobra.Command {
	lsCmd := &cobra.Command{
		Use:   "ls [<path>]",
		Short: "Lists the resources at the indicated path",
		Args:  cobra.MaximumNArgs(1),
		RunE:  toRunE(lsMain),
	}
	lsCmd.Flags().BoolP("long", "l", false, "List in long format")
	return lsCmd
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

func cname(entry apitypes.Entry) string {
	cname := entry.CName
	if entry.Supports(plugin.ListAction()) {
		cname += "/"
	}
	return cname
}

func formatLSEntries(ls []apitypes.Entry) string {
	table := make([][]string, len(ls))
	for i, entry := range ls {
		var mtimeStr string
		if entry.Attributes.HasMtime() {
			mtimeStr = format(entry.Attributes.Mtime())
		} else {
			mtimeStr = "<unknown>"
		}

		verbs := strings.Join(entry.Actions, ", ")

		table[i] = []string{cname(entry), mtimeStr, verbs}
	}
	return cmdutil.NewTableWithHeaders(headers(), table).Format()
}

func lsMain(cmd *cobra.Command, args []string) exitCode {
	// If no path is declared, try to list the current directory/resource
	path := "."
	if len(args) > 0 {
		path = args[0]
	}
	longFormat, err := cmd.Flags().GetBool("long")
	if err != nil {
		panic(err.Error())
	}

	conn := cmdutil.NewClient()
	entry, err := conn.Info(path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	var entries []apitypes.Entry
	if !entry.Supports(plugin.ListAction()) {
		entries = []apitypes.Entry{entry}
	} else {
		children, err := conn.List(path)
		if err != nil {
			cmdutil.ErrPrintf("%v\n", err)
			return exitCode{1}
		}
		entries = children
	}

	if longFormat {
		cmdutil.Print(formatLSEntries(entries))
	} else {
		for _, entry := range entries {
			cmdutil.Println(cname(entry))
		}
	}
	return exitCode{0}
}
