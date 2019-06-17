package primary

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// Path is the path primary
//
// pathPrimary => -path ShellGlob
//nolint
var Path = Parser.add(&Primary{
	Description: "Returns true if the entry's normalized path matches glob",
	name:        "path",
	args:        "glob",
	parseFunc: func(tokens []string) (*types.EntryPredicate, []string, error) {
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("requires additional arguments")
		}
		g, err := glob.Compile(tokens[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid glob: %v", err)
		}
		return types.ToEntryP(func(e types.Entry) bool {
			return g.Match(e.NormalizedPath)
		}), tokens[1:], nil
	},
})
