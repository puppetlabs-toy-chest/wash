package primary

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// pathPrimary => -path ShellGlob
//nolint
var pathPrimary = Parser.add(&primary{
	tokens: []string{"-path"},
	parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("requires additional arguments")
		}
		g, err := glob.Compile(tokens[0])
		if err != nil {
			return nil, nil, fmt.Errorf("invalid glob: %v", err)
		}
		return func(e types.Entry) bool {
			return g.Match(e.NormalizedPath)
		}, tokens[1:], nil
	},
})
