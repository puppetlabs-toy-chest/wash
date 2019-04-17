package find

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// pathPrimary => -path ShellGlob
var pathPrimary = grammar.NewAtom([]string{"-path"}, func(tokens []string) (types.Predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-path: requires additional arguments")
	}

	g, err := glob.Compile(tokens[0])
	if err != nil {
		return nil, nil, fmt.Errorf("-path: invalid glob: %v", err)
	}

	return func(e types.Entry) bool {
		return g.Match(e.NormalizedPath)
	}, tokens[1:], nil
})
