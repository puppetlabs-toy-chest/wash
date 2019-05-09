package expression

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/golang-collections/collections/stack"
)

/*
Parser is a predicate parser that parses predicate expressions. Expressions
have the following grammar:
  Expression => Expression (-a|-and) Atom |
                Expression Atom           | 
                Expression (-o|-or)  Atom |
                Atom

  Atom       => (!|-not) Atom             |
                '(' Expression ')'        |
                Predicate

where 'Expression Atom' is semantically equivalent to 'Expression -a Atom'.
The grammar for Predicate is caller-specific.

Operator precedence is (from highest to lowest):
  ()
  -not
  -and
  -or

The precedence of the () and -not operators is already enforced by the grammar.
Precedence of the binary operators -and and -or is enforced by maintaining an
evaluation stack.

Note that Parser is a sealed interface. Child classes must extend the parser
returned by NewParser when overriding the interface's methods.
*/
type Parser interface {
	predicate.Parser
	IsOp(token string) bool
	atom() *predicate.CompositeParser
	stack() *evalStack
	setStack(s *evalStack)
	insideParens() bool
	openParens()
	closeParens()
}

type parser struct {
	// Storing the binary ops this way makes it easier for us to add the capability
	// for callers to extend the parser if they'd like to support additional binary
	// ops. We will likely need this capability in the future if/when we add the ","
	// operator to `wash find`.
	binaryOps map[string]*BinaryOp
	Atom *predicate.CompositeParser
	Stack *evalStack
	numOpenParens int
	opTokens map[string]struct{}
}

// NewParser returns a new predicate expression parser. The passed-in
// predicateParser must be able to parse the "Predicate" nonterminal
// in the expression grammar.
func NewParser(predicateParser predicate.Parser) Parser {
	p := &parser{}
	p.binaryOps = make(map[string]*BinaryOp)
	p.opTokens = map[string]struct{}{
		"!": struct{}{},
		"-not": struct{}{},
		"(": struct{}{},
		")": struct{}{},
	}
	for _, op := range []*BinaryOp{andOp, orOp} {
		for _, token := range op.tokens {
			p.binaryOps[token] = op
			p.opTokens[token] = struct{}{}
		}
	}
	p.Atom = &predicate.CompositeParser{
		MatchErrMsg: "expected an atom",
		Parsers: []predicate.Parser{
			notOpParser(p),
			Parenthesize(p),
			predicateParser,
		},
	}
	return p
}

func (parser *parser) atom() *predicate.CompositeParser {
	return parser.Atom
}

func (parser *parser) stack() *evalStack {
	return parser.Stack
}

func (parser *parser) setStack(stack *evalStack) {
	parser.Stack = stack
}

func (parser *parser) insideParens() bool {
	return parser.numOpenParens > 0
}

func (parser *parser) openParens() {
	parser.numOpenParens++
}

func (parser *parser) closeParens() {
	parser.numOpenParens--
}

// IsOp returns true if the given token represents the parentheses operator,
// the not operator, or a binary operator.
func (parser *parser) IsOp(token string) bool {
	_, ok := parser.opTokens[token]
	return ok
}

/*
Parse parses a predicate expression captured by the given tokens. It will process
the tokens until it either (1) exhausts the input tokens, (2) stumbles upon a
a token that it cannot parse, or (3) finds a syntax error. For Cases (1) and (2),
Parse will return a syntax error if it did not parse a predicate. Otherwise, it will
return the parsed predicate + any remaining tokens. Case (2) will also return an
UnknownTokenError containing the offending token.

Case 2's useful if we're parsing an expression inside an expression. It lets the caller
decide if they've finished parsing the inner expression. We will take advantage of Case 2
when parsing `meta` primary expressions.

If tokens is empty, then Parse will return an ErrEmptyExpression error.
*/
func (parser *parser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	parser.setStack(newEvalStack())

	// Declare these as variables so that we can cleanly update
	// err for each iteration without having to worry about the
	// := operator's scoping rules. tks is used to avoid accidentally
	// overwriting tokens.
	//
	// POST-LOOP INVARIANT: err == nil or err is an UnknownTokenError
	var p predicate.Predicate
	var tks []string
	var err error
	for {
		// Reset err in each iteration to maintain the post-loop invariant
		err = nil
		if len(tokens) == 0 {
			break
		}
		token := tokens[0]
		if token == ")" {
			if !parser.insideParens() {
				return nil, nil, fmt.Errorf("): no beginning '('")
			}
			// We've finished parsing a parenthesized expression
			break	
		}
		// Try parsing an atom first.
		p, tks, err = parser.Atom.Parse(tokens)
		if err == nil {
			// Successfully parsed an atom, so push the parsed predicate onto the stack.
			parser.stack().pushPredicate(p)
			tokens = tks
			continue
		}
		if !errz.IsMatchError(err) {
			// Syntax error when parsing the atom, so return the error
			return nil, nil, err
		}
		// Parsing an atom didn't work, so try parsing a binaryOp
		b, ok := parser.binaryOps[token]
		if !ok {
			// Found an unknown token. Break out of the loop to evaluate
			// the final predicate.
			err = UnknownTokenError{token}
			break	
		}
		// Parsed a binaryOp, so shift tokens and push the op on the evaluation stack.
		tokens = tokens[1:]
		if parser.stack().mostRecentOp == nil {
			if _, ok := parser.stack().Peek().(predicate.Predicate); !ok {
				return nil, nil, fmt.Errorf("%v: no expression before %v", token, token)
			}
			parser.stack().pushBinaryOp(token, b)
			continue
		}
		if _, ok := parser.stack().Peek().(*BinaryOp); ok {
			// mostRecentOp's on the stack, and the parser's asking us to
			// push b. This means that mostRecentOp did not have an expression
			// after it, so report the syntax error.
			return nil, nil, fmt.Errorf(
				"%v: no expression after %v",
				parser.stack().mostRecentOpToken,
				parser.stack().mostRecentOpToken,
			)
		}
		parser.stack().pushBinaryOp(token, b)
	}
	// Parsing's finished.
	if parser.stack().Len() <= 0 {
		// We didn't parse anything. Either we have an empty expression, or
		// err is an UnknownTokenError
		if err == nil {
			err = NewEmptyExpressionError("empty expression")
		}
		// err is an UnknownTokenError
		return nil, tokens, err
	}
	if _, ok := parser.stack().Peek().(*BinaryOp); ok {
		// This codepath is possible via something like "p1 -and" or "p1 -and <unknown_token>"
		if err == nil {
			// We have "p1 -and"
			return nil, nil, fmt.Errorf(
				"%v: no expression after %v",
				parser.stack().mostRecentOpToken,
				parser.stack().mostRecentOpToken,
			)
		}
		// We have "p1 -and <unknown_token>". Pop the binary op off the stack and include
		// it as part of the remaining tokens. This is useful in case our expression is inside
		// another expression, where the top-level expression handles combining our parsed
		// predicate p with whatever's parsed by the "<unknown_token>" bit. For example, it
		// ensures that the top-level `wash find` parser correctly parses something like
		// "-m .key foo -o -m .key bar" as "Meta(.key, foo) -o Meta(.key, bar)".
		parser.stack().Pop()
		tokens = append([]string{parser.stack().mostRecentOpToken}, tokens...)
	}
	// Call s.evaluate() to handle cases like "p1 -and p2"
	parser.stack().evaluate()
	return parser.stack().Pop().(predicate.Predicate), tokens, err
}

type evalStack struct {
	*stack.Stack
	mostRecentOp *BinaryOp
	mostRecentOpToken string
}

func newEvalStack() *evalStack {
	return &evalStack{
		Stack: stack.New(),
	}
}

func (s *evalStack) pushBinaryOp(token string, b *BinaryOp) {
	// Invariant: s.Peek() returns a predicate.Predicate type.
	if s.mostRecentOp != nil {
		if b.precedence <= s.mostRecentOp.precedence {
			s.evaluate()
		}
	}
	s.mostRecentOp = b
	s.mostRecentOpToken = token
	s.Push(b)
}

func (s *evalStack) pushPredicate(p predicate.Predicate) {
	if _, ok := s.Peek().(predicate.Predicate); ok {
		// We have p1 p2, where p1 == s.Peek() and p2 = p. Since p1 p2 == p1 -and p2,
		// push andOp before pushing p2.
		s.pushBinaryOp(andOp.tokens[0], andOp)
	}
	s.Push(p)
}

func (s *evalStack) evaluate() {
	// Invariant: s's layout is something like "p (<op> p)*"
	for s.Len() > 1 {
		p2 := s.Pop().(predicate.Predicate)
		op := s.Pop().(*BinaryOp)
		p1 := s.Pop().(predicate.Predicate)
		s.Push(op.combine(p1, p2))
	}
}

