package cmdfind

import (
	"fmt"

	"github.com/gobwas/glob"
)

// pathPrimary => -path ShellGlob
var pathPrimary = newAtom([]string{"-path"}, func(tokens []string) (predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-path: requires additional arguments")
	}

	g, err := glob.Compile(tokens[0])
	if err != nil {
		return nil, nil, fmt.Errorf("-path: invalid glob: %v", err)
	}

	return func(e entry) bool {
		return g.Match(e.NormalizedPath)
	}, tokens[1:], nil
})
