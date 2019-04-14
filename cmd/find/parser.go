package cmdfind

import (
	"fmt"

	"github.com/golang-collections/collections/stack"
)

// predicate represents a predicate used by wash find
type predicate func(entry entry) bool

func parsePredicate(tokens []string) (predicate, error) {
	if len(tokens) == 0 {
		// tokens is empty, meaning the user did not provide an expression
		// to `wash find`. Thus, we default to a predicate that always returns
		// true.
		return func(e entry) bool {
			return true
		}, nil
	}
	// Validate that the parentheses are correctly balanced.
	// We do this outside of parsePredicateHelper to avoid
	// redundant validation when parensOp recurses into it.
	s := stack.New()
	for _, token := range tokens {
		if token == "(" {
			s.Push(token)
		} else if token == ")" {
			if s.Len() == 0 {
				return nil, fmt.Errorf("): no beginning '('")
			}
			s.Pop()
		}
	}
	if s.Len() > 0 {
		return nil, fmt.Errorf("(: missing closing ')'")
	}
	return parsePredicateHelper(tokens)
}

/*
An expression is described by the following grammar
	Expression => Expression (-a|-and) Atom |
	              Expression Atom           |
		      Expression (-o|-or) Atom  |
		      Atom                      |

	      Atom => (!|-not) Atom             |
	              '(' Expression ')'        |
		      Primary

where 'Expression Atom' is semantically equivalent to 'Expression -a Atom'.
Primaries have their own grammar. See the corresponding *Primary.go files
for more details.

Operator precedence is (from highest to lowest):
	()
	-not
	-and
	-or

The precedence of the () and -not operators is already enforced by the grammar.
Precedence of the binary operators -and and -or is enforced by maintaining an
evaluation stack.
*/
func parsePredicateHelper(tokens []string) (predicate, error) {
	if len(tokens) == 0 {
		panic("parsepredicate: called with len(tokens) == 0")
	}

	s := newEvalStack()
	var mostRecentOp *binaryOp
	var mostRecentOpToken string
	pushBinaryOp := func(token string, b *binaryOp) {
		// Invariant: s.Peek() returns a predicate
		if mostRecentOp != nil {
			if b.precedence <= mostRecentOp.precedence {
				s.evaluate()
			}
		}
		mostRecentOp = b
		mostRecentOpToken = token
		s.Push(b)
	}
	for len(tokens) > 0 {
		token := tokens[0]

		// Declare these as variables so that we can cleanly update the
		// tokens parameter for each iteration. Otherwise, := will create a
		// new tokens variable within the if statement's scope.
		var p predicate
		var err error
		if atom, ok := atoms[token]; ok {
			p, tokens, err = atom.parsePredicate(tokens)
			if err != nil {
				return nil, err
			}
			if _, ok := s.Peek().(predicate); ok {
				// We have p1 p2, where p1 == s.Peek() and p2 = p. Since p1 p2 == p1 -and p2,
				// push andOp before pushing p2.
				pushBinaryOp("-a", andOp)
			}
			s.Push(p)
		} else if b, ok := binaryOps[token]; ok {
			tokens = tokens[1:]
			if mostRecentOp == nil {
				if _, ok := s.Peek().(predicate); !ok {
					return nil, fmt.Errorf("%v: no expression before %v", token, token)
				}
				pushBinaryOp(token, b)
				continue
			}
			if _, ok := s.Peek().(*binaryOp); ok {
				// mostRecentOp's on the stack, and the parser's asking us to
				// push b. This means that mostRecentOp did not have an expression
				// after it, so report the error.
				return nil, fmt.Errorf("%v: no expression after %v", mostRecentOpToken, mostRecentOpToken)
			}
			pushBinaryOp(token, b)
		} else {
			return nil, fmt.Errorf("%v: unknown primary or operator", token)
		}
	}
	if _, ok := s.Peek().(*binaryOp); ok {
		// This codepath is possible via something like "p1 -and"
		return nil, fmt.Errorf("%v: no expression after %v", mostRecentOpToken, mostRecentOpToken)
	}
	// Call s.evaluate() to handle cases like "p1 -and p2"
	s.evaluate()
	return s.Pop().(predicate), nil
}

type evalStack struct {
	*stack.Stack
}

func newEvalStack() *evalStack {
	return &evalStack{&stack.Stack{}}
}

func (s *evalStack) evaluate() {
	// Invariant: s's layout is something like "p (<op> p)*"
	for s.Len() > 1 {
		p2 := s.Pop().(predicate)
		op := s.Pop().(*binaryOp)
		p1 := s.Pop().(predicate)
		s.Push(op.combine(p1, p2))
	}
}
