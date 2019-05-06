package parser

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/expression"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

/*
See the comments of expression.Parser#Parse for the grammar. Substitute
Predicate with Primary.
*/
func parseExpression(tokens []string) (predicate.Entry, error) {
	if len(tokens) == 0 {
		// tokens is empty, meaning the user did not provide an expression
		// to `wash find`. Thus, we default to a predicate that always returns
		// true.
		return func(e types.Entry) bool {
			return true
		}, nil
	}
	parser := expression.NewParser(primary.Parser)
	p, tks, err := parser.Parse(tokens)
	if err != nil {
		if tkErr, ok := err.(expression.UnknownTokenError); ok {
			err = fmt.Errorf("%v: unknown primary or operator", tkErr.Token)
		}
		return nil, err
	}
	if len(tks) != 0 {
		// This should never happen, but better safe than sorry
		msg := fmt.Sprintf("parser.parseExpression(): returned a non-empty set of tokens: %v", tks)
		panic(msg)
	}
	return p.(predicate.Entry), nil
}