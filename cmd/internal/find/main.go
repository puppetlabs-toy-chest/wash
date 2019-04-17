// Package find stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package find

import (
	"time"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

// Main is `wash find`'s main function.
func Main(cmd *cobra.Command, args []string) int {
	primary.FindStartTime = time.Now()

	result, err := parser.Parse(args)
	if err != nil {
		cmdutil.ErrPrintf("find: %v\n", err)
		return 1
	}

	conn := client.ForUNIXSocket(config.Socket)

	e, err := info(&conn, result.Path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return 1
	}
	newWalker(result, &conn).Walk(e)
	return 0
}
