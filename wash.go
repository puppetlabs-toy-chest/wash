package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/puppetlabs/wash/docker"
	"github.com/puppetlabs/wash/kubernetes"
	"github.com/puppetlabs/wash/plugin"
)

var progName = filepath.Base(os.Args[0])
var debug = flag.Bool("debug", false, "Enable debug output from FUSE")

func usage() {
	fmt.Fprintf(os.Stderr, "%s mounts remote resources with FUSE", progName)
	fmt.Fprintf(os.Stderr, "Usage: %s MOUNTPOINT\n", progName)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	log.SetFlags(log.Ldate | log.Lmicroseconds)

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	if err := mount(mountpoint); err != nil {
		log.Fatal(err)
	}
}

type clientInit struct {
	name   string
	client plugin.DirProtocol
	err    error
}

type instantiator = func(string, bool) (plugin.DirProtocol, error)

func mount(mountpoint string) error {
	clients := make(chan clientInit)

	if *debug {
		fuse.Debug = func(msg interface{}) {
			log.Println(msg)
		}
	}

	clientInstantiators := map[string]instantiator{
		"docker":     docker.Create,
		"kubernetes": kubernetes.Create,
	}

	for k, v := range clientInstantiators {
		go func(name string, create instantiator) {
			log.Printf("Loading %v integration", name)
			client, err := create(name, *debug)
			clients <- clientInit{name, client, err}
		}(k, v)
	}

	log.Println("Mounting at", mountpoint)
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	clientMap := make(map[string]plugin.DirProtocol)
	for range clientInstantiators {
		client := <-clients
		if client.err != nil {
			return client.err
		}
		clientMap[client.name] = client.client
	}

	log.Println("Serving filesystem")
	filesys := &plugin.FS{Clients: clientMap}
	if err := fs.Serve(c, filesys); err != nil {
		return err
	}

	// check if the mount process has an error to report
	<-c.Ready
	if err := c.MountError; err != nil {
		return err
	}
	log.Println("Done")

	return nil
}
