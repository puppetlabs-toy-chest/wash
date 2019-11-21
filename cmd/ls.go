package cmd

import (
	"bytes"
	"time"

	"github.com/spf13/cobra"

	apitypes "github.com/puppetlabs/wash/api/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

func lsCommand() *cobra.Command {
	var perms string
	for _, entry := range permMap {
		perms += "  " + entry.name + " => " + string(entry.short) + "\n"
	}
	lsCmd := &cobra.Command{
		Use:   "ls [<path>]",
		Short: "Lists resources at the indicated path",
		Long: `Lists resources at the indicated path.

When using the long format, permissions are abreviated:
` + perms,
		Args: cobra.MaximumNArgs(1),
		RunE: toRunE(lsMain),
	}
	lsCmd.Flags().BoolP("long", "l", false, "List in long format")
	return lsCmd
}

func headers() []cmdutil.ColumnHeader {
	return []cmdutil.ColumnHeader{
		{ShortName: "name", FullName: "NAME"},
		{ShortName: "crtime", FullName: "CREATED"},
		{ShortName: "mtime", FullName: "MODIFIED"},
		{ShortName: "perms", FullName: "PERMISSIONS"},
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

type permEntry struct {
	name  string
	short byte
}

var permMap = []permEntry{{"list", 'l'}, {"read", 'r'}, {"write", 'w'}, {"exec", 'x'}, {"stream", 's'}, {"delete", 'd'}}

func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func formatLSEntries(ls []apitypes.Entry) string {
	table := make([][]string, len(ls))
	for i, entry := range ls {
		var crtimeStr, mtimeStr string
		if entry.Attributes.HasMtime() {
			mtimeStr = format(entry.Attributes.Mtime())
		} else {
			mtimeStr = "<unknown>"
		}

		if entry.Attributes.HasCrtime() {
			crtimeStr = format(entry.Attributes.Crtime())
		} else {
			crtimeStr = "<unknown>"
		}

		// Permissions are: lrwxsd (list, read, write, exec, stream, delete)
		perms := bytes.Repeat([]byte{'-'}, len(permMap))
		for i, perm := range permMap {
			if contains(entry.Actions, perm.name) {
				perms[i] = perm.short
			}
		}

		table[i] = []string{cname(entry), crtimeStr, mtimeStr, string(perms)}
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
