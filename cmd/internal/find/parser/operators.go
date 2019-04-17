package parser

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

//nolint
var notOp = grammar.NewAtom([]string{"!", "-not"}, func(tokens []string) (types.Predicate, []string, error) {
	notToken := tokens[0]
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("%v: no following expression", notToken)
	}
	token := tokens[0]
	if atom, ok := grammar.Atoms[token]; ok {
		p, tokens, err := atom.Parse(tokens)
		if err != nil {
			return nil, nil, err
		}
		return func(e types.Entry) bool {
			return !p(e)
		}, tokens, err
	}
	return nil, nil, fmt.Errorf("%v: no following expression", notToken)
})

//nolint
var parensOp = grammar.NewAtom([]string{"("}, func(tokens []string) (types.Predicate, []string, error) {
	// Find the ")" that's paired with our "(". Use the algorithm
	// described in https://stackoverflow.com/questions/12752225/how-do-i-find-the-position-of-matching-parentheses-or-braces-in-a-given-piece-of
	// Note that we do not have to check for balanced parentheses here because
	// that check was already done in the top-level parse method.
	//
	// TODO: Could optimize this by moving parens handling over to the evaluation
	// stack. Not important right now because expressions to `wash find` will likely
	// be simple.
	tokens = tokens[1:]
	ix := 0
	counter := 1
	for i, token := range tokens {
		if token == "(" {
			counter++
		} else if token == ")" {
			counter--
			ix = i
		}
		if counter == 0 {
			break
		}
	}
	if ix == 0 {
		return nil, nil, fmt.Errorf("(): empty inner expression")
	}
	p, err := parseExpressionHelper(tokens[:ix])
	return p, tokens[ix+1:], err
})

//nolint
var andOp = grammar.NewBinaryOp([]string{"-a", "-and"}, 1, func(p1 types.Predicate, p2 types.Predicate) types.Predicate {
	return func(e types.Entry) bool {
		return p1(e) && p2(e)
	}
})

//nolint
var orOp = grammar.NewBinaryOp([]string{"-o", "-or"}, 0, func(p1 types.Predicate, p2 types.Predicate) types.Predicate {
	return func(e types.Entry) bool {
		return p1(e) || p2(e)
	}
})
