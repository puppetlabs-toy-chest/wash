// Package find stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package find

import (
	"strings"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

// Main is `wash find`'s main function.
func Main(args []string) int {
	params.ReferenceTime = time.Now()

	// Parse the arguments
	result, err := parser.Parse(args)
	opts := &result.Options
	if opts.Help.Requested {
		return printHelp(result.Options.Help)
	}
	if err != nil {
		cmdutil.ErrPrintf("find: %v\n", err)
		return 1
	}
	if opts.Daystart {
		// Set the ReferenceTime to the start of the current day
		year, month, day := params.ReferenceTime.Date()
		params.ReferenceTime = time.Date(
			year,
			month,
			day,
			0,
			0,
			0,
			0,
			params.ReferenceTime.Location(),
		)
	}

	// Do the walk
	conn := cmdutil.NewClient()
	walker := newWalker(result, conn)
	exitCode := 0
	for _, path := range result.Paths {
		if !walker.Walk(path) {
			exitCode = 1
		}
	}
	return exitCode
}

func printHelp(helpOpt types.HelpOption) int {
	printDescription := func(desc string) {
		desc = strings.Trim(desc, "\n")
		cmdutil.Println(desc)
	}
	if !helpOpt.HasValue {
		cmdutil.Print(Usage())
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
