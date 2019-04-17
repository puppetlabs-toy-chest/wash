package find

import (
	"fmt"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// namePrimary => -name ShellGlob
var namePrimary = grammar.NewAtom([]string{"-name"}, func(tokens []string) (types.Predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-name: requires additional arguments")
	}

	g, err := glob.Compile(tokens[0])
	if err != nil {
		return nil, nil, fmt.Errorf("-name: invalid glob: %v", err)
	}

	return func(e types.Entry) bool {
		return g.Match(e.CName)
	}, tokens[1:], nil
})
