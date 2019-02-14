package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/InVisionApp/tabular"
	"github.com/pkg/xattr"

	"github.com/puppetlabs/wash/api/client"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "%s prints all extended attributes of a file as a YAML object\n", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", progName)
	flag.PrintDefaults()
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
	var tab tabular.Table

	// Setup the output table
	tab = tabular.New()
	nameWidth := len(longestFieldFromListing(ls, func(e client.LSItem) string {
		return e.Name
	}))
	verbsWidth := len(longestFieldFromListing(ls, func(e client.LSItem) string {
		return strings.Join(e.Commands, ", ")
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

		cmds := entry.Commands
		sort.Strings(cmds)
		verbs := strings.Join(cmds, ", ")

		isDir := cmds[sort.SearchStrings(cmds, "list")] == "list"
		if isDir {
			name += "/"
		}

		out += fmt.Sprintf(table.Format, name, ctime, verbs)
	}
	return out
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	path := flag.Arg(0)
	apiPath, err := xattr.Get(path, "wash.id")
	if err != nil {
		log.Fatal(err)
	}

	conn := client.ClientUNIXSocket("/tmp/wash-api.sock")
	ls, err := client.List(conn, string(apiPath))

	if err != nil {
		panic("Fatal error")
	}

	fmt.Print(formatTabularListing(ls))
}
