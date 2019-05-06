package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

/*
Predicate => ObjectPredicate |
             ArrayPredicate  |
             PrimitivePredicate
*/
func parsePredicate(tokens []string) (predicate.Generic, []string, error) {
	cp := &predicate.CompositeParser{
		ErrMsg: "expected either a primitive, object, or array predicate",
		Parsers: []predicate.Parser{
			predicate.GenericParser(parseObjectPredicate),
			predicate.GenericParser(parseArrayPredicate),
			predicate.GenericParser(parsePrimitivePredicate),
		},
	}
	p, tokens, err := cp.Parse(tokens)
	if err != nil {
		return nil, nil, err
	}
	return p.(predicate.Generic), tokens, err
}
