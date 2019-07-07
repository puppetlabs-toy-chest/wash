package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/expression"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

type predicateExpressionParser struct {
	expression.Parser
	isInnerExpression bool
}

func newPredicateExpressionParser(isInnerExpression bool) predicate.Parser {
	p := &predicateExpressionParser{
		Parser:            expression.NewParser(predicate.ToParser(parsePredicate), &predicateAnd{}, &predicateOr{}),
		isInnerExpression: isInnerExpression,
	}
	p.SetEmptyExpressionErrMsg("expected a predicate expression")
	p.SetUnknownTokenErrFunc(func(token string) string {
		return fmt.Sprintf("unknown predicate %v", token)
	})
	if isInnerExpression {
		// Inner expressions are parenthesized so that they do not conflict with
		// the top-level predicate expression parser.
		return expression.Parenthesize(p)
	}
	return p
}
