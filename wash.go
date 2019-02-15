package main

import (
	"fmt"
	"log"
	"os"

	"github.com/puppetlabs/wash/cmd"
	"github.com/puppetlabs/wash/config"
)

func main() {
	if err := config.Load(); err != nil {
		log.Fatal(fmt.Sprintf("Failed to load Wash's config: %v", err))
		os.Exit(1)
	}

	os.Exit(cmd.Execute())
}
