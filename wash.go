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

	if flag.NArg() != 1 {
		usage()
		os.Exit(2)
	}
	mountpoint := flag.Arg(0)
	if err := mount(mountpoint); err != nil {
		log.Fatal(err)
	}
}

func mount(mountpoint string) error {
	log.Println("Loading docker integration")
	dockercli, err := docker.Create(*debug)
	if err != nil {
		return err
	}

	if *debug {
		fuse.Debug = func(msg interface{}) {
			log.Println(msg)
		}
	}

	log.Println("Mounting at", mountpoint)
	c, err := fuse.Mount(mountpoint)
	if err != nil {
		return err
	}
	defer c.Close()

	log.Println("Serving filesystem with docker")
	filesys := &plugin.FS{
		Clients: map[string]plugin.DirProtocol{
			"docker": dockercli,
		},
	}
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
