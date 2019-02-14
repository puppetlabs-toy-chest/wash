package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/InVisionApp/tabular"
	"github.com/pkg/xattr"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "%s prints all extended attributes of a file as a YAML object\n", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", progName)
	flag.PrintDefaults()
}

// TODO: consider moving this into the api package
type listingEntry struct {
	Commands   []string `json:"commands"`
	Name       string   `json:"name"`
	Attributes struct {
		Atime string `json:"Atime"`
		Mtime string `json:"Mtime"`
		Ctime string `json:"Ctime"`
		Mode  int    `json:"Mode"`
		Size  int    `json:"Size"`
		Valid int    `json:"Valid"`
	} `json:"attributes"`
}

type listing []listingEntry

func api(client http.Client, command string, path string) ([]byte, error) {
	url := fmt.Sprintf("http://localhost/fs/%s%s", command, path)
	response, err := client.Get(url)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if response.StatusCode != http.StatusOK {
		// Generate a real error object for this
		log.Printf("Status: %v, Body: %v", response.StatusCode, string(body))
		return nil, err
	}

	return body, nil
}

func getListing(client http.Client, path string) (listing, error) {
	body, err := api(client, "list", path)
	if err != nil {
		return nil, err
	}

	var ls listing
	if err := json.Unmarshal(body, &ls); err != nil {
		return nil, err
	}

	return ls, nil
}

func longestFieldFromListing(ls listing, lookup func(listingEntry) string) string {
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

func formatTabularListing(ls listing) string {
	var out string
	var tab tabular.Table

	// Setup the output table
	tab = tabular.New()
	nameWidth := len(longestFieldFromListing(ls, func(e listingEntry) string {
		return e.Name
	}))
	verbsWidth := len(longestFieldFromListing(ls, func(e listingEntry) string {
		return strings.Join(e.Commands, ", ")
	}))
	tab.Col("size", "NAME", nameWidth+2)
	tab.Col("ctime", "CREATED", 25+2)
	tab.Col("verbs", "ACTIONS", verbsWidth+2)

	table := tab.Parse("*")
	out += fmt.Sprintln(table.Header)

	for _, entry := range ls {
		name := entry.Name

		ctime := entry.Attributes.Ctime
		if ctime == "" {
			ctime = "<unknown>"
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

	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/tmp/wash-api.sock")
			},
		},
	}

	ls, err := getListing(httpc, string(apiPath))
	if err != nil {
		panic("Fatal error")
	}

	fmt.Print(formatTabularListing(ls))
}
