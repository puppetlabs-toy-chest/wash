// Package find stores all the logic for `wash find`. We make it a separate package
// to decouple it from cmd. This makes testing easier.
package find

import (
	"fmt"
	"time"

	"github.com/golang-collections/collections/stack"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/config"
	"github.com/puppetlabs/wash/plugin"
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
	path := result.Path
	p := result.Predicate

	conn := client.ForUNIXSocket(config.Socket)

	e, err := info(&conn, path)
	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return 1
	}

	s := stack.New()
	s.Push(e)
	for s.Len() > 0 {
		e := s.Pop().(types.Entry)
		if p(e) {
			fmt.Printf("%v\n", e.NormalizedPath)
		}
		if e.Supports(plugin.ListAction) {
			children, err := list(&conn, e)
			if err != nil {
				cmdutil.ErrPrintf("could not get children of %v: %v\n", e.NormalizedPath, err)
			}
			for _, child := range children {
				s.Push(child)
			}
		}
	}
	return 0
}
