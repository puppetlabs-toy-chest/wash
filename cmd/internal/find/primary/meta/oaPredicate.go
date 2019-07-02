package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// parseOAPredicate is tested in the tests for ObjectPredicate/ArrayPredicate

func parseOAPredicate(tokens []string) (Predicate, []string, error) {
	cp := &predicate.CompositeParser{
		MatchErrMsg: "expected a predicate or a parenthesized predicate expression",
		Parsers: []predicate.Parser{
			toPredicateParser(parsePredicate),
			newPredicateExpressionParser(true),
		},
	}
	p, tokens, err := cp.Parse(tokens)
	if err != nil {
		return nil, tokens, err
	}
	return p.(Predicate), tokens, err
}
