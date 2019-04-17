package find

import (
	"fmt"

	"github.com/gobwas/glob"
)

// namePrimary => -name ShellGlob
var namePrimary = newAtom([]string{"-name"}, func(tokens []string) (predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-name: requires additional arguments")
	}

	g, err := glob.Compile(tokens[0])
	if err != nil {
		return nil, nil, fmt.Errorf("-name: invalid glob: %v", err)
	}

	return func(e entry) bool {
		return g.Match(e.CName)
	}, tokens[1:], nil
})
