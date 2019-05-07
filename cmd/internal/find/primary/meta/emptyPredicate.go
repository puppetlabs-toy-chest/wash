package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
)

// emptyPredicate => -empty
func parseEmptyPredicate(tokens []string) (Predicate, []string, error) {
	if len(tokens) == 0 || tokens[0] != "-empty" {
		return nil, nil, errz.NewMatchError("expected '-empty'")
	}
	return emptyP, tokens[1:], nil
}

func emptyP(v interface{}) bool {
	switch t := v.(type) {
	case map[string]interface{}:
		return len(t) == 0
	case []interface{}:
		return len(t) == 0
	default:
		return false
	}
}
