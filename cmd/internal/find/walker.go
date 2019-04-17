package find

import (
	"fmt"

	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
)

type walker struct {
	p    types.Predicate
	opts types.Options
	conn *client.DomainSocketClient
}

func newWalker(r parser.Result, conn *client.DomainSocketClient) *walker {
	return &walker{
		p:    r.Predicate,
		opts: r.Options,
		conn: conn,
	}
}

func (w *walker) Walk(e types.Entry) {
	w.walk(e, 0)
}

func (w *walker) walk(e types.Entry, depth uint) {
	// If the Depth option is set, then we visit e after visiting its children.
	// Otherwise, we visit e first.
	//
	// TODO: Write unit tests for the walker by mocking out the client.
	if !w.opts.Depth {
		w.visit(e, depth)
	}
	childDepth := depth + 1
	if childDepth <= w.opts.Maxdepth && e.Supports(plugin.ListAction) {
		children, err := list(w.conn, e)
		if err != nil {
			cmdutil.ErrPrintf("could not get children of %v: %v\n", e.NormalizedPath, err)
		} else {
			for _, child := range children {
				w.walk(child, childDepth)
			}
		}
	}
	if w.opts.Depth {
		w.visit(e, depth)
	}
}

func (w *walker) visit(e types.Entry, depth uint) {
	if depth >= w.opts.Mindepth && w.p(e) {
		fmt.Printf("%v\n", e.NormalizedPath)
	}
}
