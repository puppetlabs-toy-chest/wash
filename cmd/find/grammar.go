package cmdfind

import (
	"fmt"

	apitypes "github.com/puppetlabs/wash/api/types"
)

type atom struct {
	tokens []string
	// tokens[0] will always include the atom's token that the user
	// passed-in
	parsePredicate func(tokens []string) (Predicate, []string, error)
}

var allAtoms = []*atom{
	notOp,
	parensOp,
	namePrimary,
	truePrimary,
	falsePrimary,
}

// Map of <token> => <atom>. This is initialized in ParsePredicate. Unfortunately,
// we cannot intialize atoms here because doing so would result in an initialization
// loop compiler error (e.g. notOp => atoms => notOp).
var atoms = make(map[string]*atom)

var notOp = &atom{
	tokens: []string{"!", "-not"},
	parsePredicate: func(tokens []string) (Predicate, []string, error) {
		notToken := tokens[0]
		tokens = tokens[1:]
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("%v: no following expression", notToken)
		}
		token := tokens[0]
		if atom, ok := atoms[token]; ok {
			p, tokens, err := atom.parsePredicate(tokens)
			if err != nil {
				return nil, nil, err
			}
			return func(e *apitypes.ListEntry) bool {
				return !p(e)
			}, tokens, err
		}
		return nil, nil, fmt.Errorf("%v: no following expression", notToken)
	},
}

var parensOp = &atom{
	tokens: []string{"("},
	parsePredicate: func(tokens []string) (Predicate, []string, error) {
		// Find the ")" that's paired with our "(". Use the algorithm
		// described in https://stackoverflow.com/questions/12752225/how-do-i-find-the-position-of-matching-parentheses-or-braces-in-a-given-piece-of
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
		p, err := parsePredicate(tokens[:ix])
		return p, tokens[ix+1:], err
	},
}

type binaryOp struct {
	tokens     []string
	precedence int
	combine    func(p1 Predicate, p2 Predicate) Predicate
}

var allBinaryOps = []*binaryOp{
	andOp,
	orOp,
}

// Map of <token> => <binaryOp>
var binaryOps = make(map[string]*binaryOp)

var andOp = &binaryOp{
	tokens:     []string{"-a", "-and"},
	precedence: 1,
	combine: func(p1 Predicate, p2 Predicate) Predicate {
		return func(e *apitypes.ListEntry) bool {
			return p1(e) && p2(e)
		}
	},
}

var orOp = &binaryOp{
	tokens:     []string{"-o", "-or"},
	precedence: 0,
	combine: func(p1 Predicate, p2 Predicate) Predicate {
		return func(e *apitypes.ListEntry) bool {
			return p1(e) || p2(e)
		}
	},
}
