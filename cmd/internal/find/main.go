// Package find stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package find

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/cmd/internal/config"
	"github.com/spf13/cobra"
)

// Main is `wash find`'s main function.
func Main(cmd *cobra.Command, args []string) int {
	params.ReferenceTime = time.Now()

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
	if primary.IsSet(primary.Meta) && !opts.IsSet(types.MaxdepthFlag) {
		// The `meta` primary's a specialized filter. It should only be used
		// if a user needs to filter on something that isn't in plugin.EntryAttributes
		// (e.g. like an EC2 instance tag, a Docker container's image, etc.). Thus, it
		// wouldn't make sense for `wash find` to recurse when someone's using the `meta`
		// primary since it is likely that siblings or children will have a different meta
		// schema. For example, if we're filtering EC2 instances based on a tag, then `wash find`
		// shouldn't recurse down into the EC2 instance's console output + metadata.json files
		// because those entries don't have tags and, even if they did, they'd likely be under a
		// different key (e.g. like "Labels" for Docker containers). Thus to avoid the unnecessary
		// recursion, we default maxdepth to 1 if the flag was not set by the user. Note that users
		// who want to recurse down into subdirectories can just set maxdepth to -1. The recursion
		// is useful when running `wash find` inside a directory whose entries and subdirectory entries
		// all have the same `meta` schema (e.g. like in an S3 bucket).
		fmt.Fprintln(os.Stderr, "The meta primary is being used. Setting maxdepth to 1...")
		opts.Maxdepth = 1
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
