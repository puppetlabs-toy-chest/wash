// Source for the wash executable.
//
// To extend wash, see documentation for the 'plugin' package.
package main

import (
	"os"

	"github.com/puppetlabs/wash/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
