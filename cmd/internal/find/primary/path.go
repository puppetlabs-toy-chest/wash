package primary

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// pathPrimary => -path ShellGlob
//nolint
var pathPrimary = Parser.newPrimary([]string{"-path"}, func(tokens []string) (predicate.Entry, []string, error) {
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
})
