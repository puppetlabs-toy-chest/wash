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
	p, tokens, err := try(
		tokens,
		parseObjectPredicate,
		parseArrayPredicate,
		parsePrimitivePredicate,
	)
	if err != nil {
		if errz.IsMatchError(err) {
			return nil, nil, errz.NewMatchError("expected either a primitive, object, or array predicate")
		}
		return nil, nil, err
	}
	return p, tokens, err
}
