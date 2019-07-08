package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// parseOAPredicate is tested in the tests for ObjectPredicate/ArrayPredicate

func parseOAPredicate(tokens []string) (predicate.Predicate, []string, error) {
	cp := &predicate.CompositeParser{
		MatchErrMsg: "expected a predicate or a parenthesized predicate expression",
		Parsers: []predicate.Parser{
			predicate.ToParser(parsePredicate),
			newPredicateExpressionParser(true),
		},
	}
	return cp.Parse(tokens)
}
