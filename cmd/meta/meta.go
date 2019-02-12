package main

import (
	"bytes"
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

	"github.com/pkg/xattr"
)

var progName = filepath.Base(os.Args[0])

func usage() {
	fmt.Fprintf(os.Stderr, "%s prints all extended attributes of a file as a YAML object\n", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", progName)
	flag.PrintDefaults()
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

	url := fmt.Sprintf("http://localhost/fs/metadata%v", string(apiPath))
	response, err := httpc.Get(url)
	if err != nil {
		log.Fatal(err)
		return
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
		return
	}

	if response.StatusCode != http.StatusOK {
		log.Fatal(fmt.Sprintf("Status: %v, Body: %v", response.StatusCode, string(body)))
		return
	}

	var metadataBuffer bytes.Buffer
	json.Indent(&metadataBuffer, body, "", "  ")

	metadataBuffer.WriteTo(os.Stdout)
}
