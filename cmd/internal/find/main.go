// Package find stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package find

import (
	"fmt"
	"strings"
	"time"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/spf13/cobra"
)

// Main is `wash find`'s main function.
func Main(cmd *cobra.Command, args []string) int {
	params.StartTime = time.Now()

	result, err := parser.Parse(args)
	if result.Options.Help.Requested {
		return printHelp(result.Options.Help)
	}
	if err != nil {
		cmdutil.ErrPrintf("find: %v\n", err)
		return 1
	}

	conn := client.ForUNIXSocket(config.Socket)

	e, err := info(conn, result.Path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return 1
	}
	newWalker(result, conn).Walk(e)
	return 0
}

func printHelp(helpOpt types.HelpOption) int {
	printDescription := func(desc string) {
		desc = strings.Trim(desc, "\n")
		fmt.Println(desc)
	}
	if !helpOpt.HasValue {
		fmt.Print(Usage())
	} else if helpOpt.Syntax {
		printDescription(parser.ExpressionSyntaxDescription)
	} else {
		p := primary.Get(helpOpt.Primary)
		if p == nil {
			cmdutil.ErrPrintf("unknown primary %v", helpOpt.Primary)
			return 1
		}
		desc := p.DetailedDescription
		if desc == "" {
			desc = p.Usage() + "\n" + p.Description
		}
		printDescription(desc)
	}
	return 0
}
