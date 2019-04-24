package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

// predicate represents a predicate in the meta primary's grammar.
type predicate func(v interface{}) bool

/*
Predicate => ObjectPredicate |
             ArrayPredicate  |
             PrimitivePredicate
*/
func parsePredicate(tokens []string) (predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected either a primitive, object, or array predicate")
	}
	switch token := tokens[0]; token[0] {
	case '.':
		return parseObjectPredicate(tokens)
	case '[':
		fallthrough
	case ']':
		return parseArrayPredicate(tokens)
	default:
		return parsePrimitivePredicate(tokens)
	}
}
