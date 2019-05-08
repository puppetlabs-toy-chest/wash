package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/expression"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// PredicateExpression => (See the comments of expression.Parser#Parse)
func parsePredicateExpression(tokens []string) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("expected a predicate expression")
	}
	parser := expression.NewParser(predicate.ToParser(parsePredicate))
	p, tks, err := parser.Parse(tokens)
	if err != nil {
		tkErr, ok := err.(expression.UnknownTokenError)
		if !ok {
			// We have a syntax error
			return nil, nil, err
		}
		if p == nil {
			// possible via something like "-size + 1"
			return nil, nil, fmt.Errorf("unknown predicate %v", tkErr.Token)
		}
	}
	// If err != nil here, then err is an UnknownTokenError. An UnknownTokenError
	// means we've finished parsing the `meta` primary's predicate expression.
	return p, tks, nil
}