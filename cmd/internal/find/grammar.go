package find

import (
	"fmt"
)

type atom struct {
	tokens []string
	// tokens[0] will always include the atom's token that the user
	// passed-in
	parse func(tokens []string) (predicate, []string, error)
}

// Map of <token> => <atom>. This is populated by newAtom.
var atoms = make(map[string]*atom)

// When creating a new atom with this function, be sure to comment nolint above the variable
// so that CI does not mark it as unused. See notOp for an example.
func newAtom(tokens []string, parse func(tokens []string) (predicate, []string, error)) *atom {
	a := &atom{
		tokens: tokens,
		parse:  parse,
	}
	for _, t := range tokens {
		atoms[t] = a
	}
	return a
}

//nolint
var notOp = newAtom([]string{"!", "-not"}, func(tokens []string) (predicate, []string, error) {
	notToken := tokens[0]
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("%v: no following expression", notToken)
	}
	token := tokens[0]
	if atom, ok := atoms[token]; ok {
		p, tokens, err := atom.parse(tokens)
		if err != nil {
			return nil, nil, err
		}
		return func(e entry) bool {
			return !p(e)
		}, tokens, err
	}
	return nil, nil, fmt.Errorf("%v: no following expression", notToken)
})

//nolint
var parensOp = newAtom([]string{"("}, func(tokens []string) (predicate, []string, error) {
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
	p, err := parseExpression(tokens[:ix])
	return p, tokens[ix+1:], err
})

type binaryOp struct {
	tokens     []string
	precedence int
	combine    func(p1 predicate, p2 predicate) predicate
}

// Map of <token> => <binaryOp>. This is populated by newBinaryOp.
var binaryOps = make(map[string]*binaryOp)

// When creating a new binary op with this function, be sure to comment nolint above the variable
// so that CI does not mark it as unused. See andOp for an example.
func newBinaryOp(tokens []string, precedence int, combine func(p1 predicate, p2 predicate) predicate) *binaryOp {
	b := &binaryOp{
		tokens:     tokens,
		precedence: precedence,
		combine:    combine,
	}
	for _, t := range tokens {
		binaryOps[t] = b
	}
	return b
}

//nolint
var andOp = newBinaryOp([]string{"-a", "-and"}, 1, func(p1 predicate, p2 predicate) predicate {
	return func(e entry) bool {
		return p1(e) && p2(e)
	}
})

//nolint
var orOp = newBinaryOp([]string{"-o", "-or"}, 0, func(p1 predicate, p2 predicate) predicate {
	return func(e entry) bool {
		return p1(e) || p2(e)
	}
})
