package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-yaml/yaml"
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

	var err error
	var list []string
	if list, err = xattr.List(path); err != nil {
		log.Fatal(err)
	}

	sort.Strings(list)
	everything := make(map[string]interface{})
	for _, key := range list {
		var data []byte
		if data, err = xattr.Get(path, key); err != nil {
			log.Fatal(err)
		}

		var structured interface{}
		if err = json.Unmarshal(data, &structured); err != nil {
			everything[key] = string(data)
		} else {
			everything[key] = structured
		}
	}
	if b, err := yaml.Marshal(everything); err != nil {
		log.Fatal(err)
	} else {
		fmt.Println(string(b))
	}
}
