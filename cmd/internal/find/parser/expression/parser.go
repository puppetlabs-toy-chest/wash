package expression

import (
	"fmt"

	"github.com/golang-collections/collections/stack"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
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
	// SetUnknownTokenErrFunc sets a function to generate an error
	// message when the parser encounters an unknown token.
	SetUnknownTokenErrFunc(func(string) string)
	// SetEmptyExpressionErrMsg sets the error message that will be used
	// when the parser encounters an empty expression.
	SetEmptyExpressionErrMsg(string)
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
	binaryOps             map[string]*BinaryOp
	Atom                  *predicate.CompositeParser
	Stack                 *evalStack
	numOpenParens         int
	opTokens              map[string]struct{}
	unknownTokenErrFunc   func(string) string
	emptyExpressionErrMsg string
}

// NewParser returns a new predicate expression parser. The passed-in
// predicateParser must be able to parse the "Predicate" nonterminal
// in the expression grammar.
func NewParser(predicateParser predicate.Parser, andOp predicate.BinaryOp, orOp predicate.BinaryOp) Parser {
	p := &parser{}
	p.binaryOps = make(map[string]*BinaryOp)
	p.opTokens = map[string]struct{}{
		"!":    struct{}{},
		"-not": struct{}{},
		"(":    struct{}{},
		")":    struct{}{},
	}
	for _, op := range []*BinaryOp{newAndOp(andOp), newOrOp(orOp)} {
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
	p.SetEmptyExpressionErrMsg("empty expression")
	p.SetUnknownTokenErrFunc(func(token string) string {
		return fmt.Sprintf("unknown token %v", token)
	})
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

func (parser *parser) SetUnknownTokenErrFunc(errFunc func(string) string) {
	if errFunc == nil {
		panic("parser.SetUnknownTokenErrFunc called with a nil errFunc!")
	}
	parser.unknownTokenErrFunc = errFunc
}

func (parser *parser) SetEmptyExpressionErrMsg(msg string) {
	if len(msg) <= 0 {
		panic("parser.SetEmptyExpressionErrMsg called with an empty msg")
	}
	parser.emptyExpressionErrMsg = msg
}

/*
Parse parses a predicate expression captured by the given tokens. It will process
the tokens until it either:
	1. Exhausts the input tokens
	2. Stumbles upon an unknown token (token that it cannot parse)
	3. Stumbles upon an incomplete operator (i.e. a dangling ")" or a "!" operator)
	4. Finds a syntax error
For Cases (1), (2), and (3), Parse will return a syntax error if it did not parse a
predicate. Otherwise, it will return the parsed predicate + any remaining tokens.
Case (2) will also return an errz.UnknownTokenError containing the offending token,
while Case (3) will return an errz.IncompleteOperatorError.

If you're using the expression parser, then instead of the usual

	p, tokens, err := expression.NewParser(...).Parse(tokens)
	if err != nil {
		// Optional code to wrap the error //
		return nil, nil, err
	}
	return p, tokens, err

the following pattern's recommended:

	p, tokens, err := expression.NewParser(...).Parse(tokens)
	if err != nil {
		// Optional code to wrap the error    //
		// Set the error to the wrapped error //
	}
	return p, tokens, err

This pattern makes it easy for the expression parser to handle parsing nested
predicate expressions without burdening the caller with that responsibility. We
take advantage of this pattern when parsing meta primary predicate expressions.

NOTE: If an unknown token/incomplete operator is encountered inside a parenthesized
expression, then a syntax error is returned. The reason for this decision is because
parenthesized expressions have their own context (they are their own inner expression).
Hence, they can handle their unknown token/incomplete operator errors. However,
non-parenthesized expressions could be embedded as part of an outer expression. In the
latter case, the outer expression's parser would handle the error.
*/
func (parser *parser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	parser.setStack(newEvalStack(parser.binaryOps["-a"]))

	// Declare these as variables so that we can cleanly update
	// err for each iteration without having to worry about the
	// := operator's scoping rules. tks is used to avoid accidentally
	// overwriting tokens.
	//
	// POST-LOOP INVARIANT: err == nil or err is an UnknownTokenError/IncompleteOperatorError
	var p predicate.Predicate
	var tks []string
	var err error
Loop:
	for {
		// Reset err in each iteration to maintain the post-loop invariant
		err = nil
		if len(tokens) == 0 {
			break
		}
		token := tokens[0]
		if token == ")" {
			if !parser.insideParens() {
				err = errz.IncompleteOperatorError{
					Reason: "): no beginning '('",
				}
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
			if errz.IsSyntaxError(err) {
				return nil, nil, err
			}
			if p != nil {
				// Push the parsed predicate onto the stack, then set tokens to tks and reset
				// the error so that we (the callers) handle it in the next iteration.
				parser.stack().pushPredicate(p)
				tokens = tks
				err = nil
				continue
			}
			// p == nil
			switch err.(type) {
			case errz.UnknownTokenError:
				msg := fmt.Sprintf("parser.Parse: an atom returned an unknown token error without parsing a predicate: %v", err)
				panic(msg)
			case errz.IncompleteOperatorError:
				// A predicate wasn't parsed. This is possible via something like
				// "-m .key -exists -a ! -name foo" where the "!" would return this
				// error because "-name" is not a valid meta primary expression.
				//
				// If we hit this case, that means parsing's finished. Thus, we break
				// out of the loop and let our caller handle the IncompleteOperatorError.
				// Note that in our example, this would mean that `wash find`'s top-level
				// expression parser would handle the "! -name foo" part of the expression,
				// which is correct.
				break Loop
			default:
				// We should never hit this code-path
				msg := fmt.Sprintf("Unknown error %v", err)
				panic(msg)
			}
		}
		// Parsing an atom didn't work, so try parsing a binaryOp
		b, ok := parser.binaryOps[token]
		if !ok {
			// Found an unknown token. Break out of the loop to evaluate
			// the final predicate.
			err = errz.UnknownTokenError{
				Token: token,
				Msg:   parser.unknownTokenErrFunc(token),
			}
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
		// err is an UnknownTokenError/IncompleteOperatorError. In either case,
		// this is considered a syntax error.
		if err == nil {
			err = emptyExpressionError{
				parser.emptyExpressionErrMsg,
			}
		} else {
			// err is an UnknownTokenError/IncompleteOperatorError
			err = fmt.Errorf(err.Error())
		}
		return nil, tokens, err
	}
	if parser.insideParens() && err != nil {
		// We have an UnknownTokenError/IncompleteOperatorError inside a parenthesized
		// expression. Since a parenthesized expression is its own context, this is
		// considered a syntax error.
		err = fmt.Errorf(err.Error())
		return nil, tokens, err
	}
	if _, ok := parser.stack().Peek().(*BinaryOp); ok {
		// This codepath is possible via something like "p1 -and" or
		// "p1 -and <unknown_token>/<incomplete_operator>"
		if err == nil {
			// We have "p1 -and"
			return nil, nil, fmt.Errorf(
				"%v: no expression after %v",
				parser.stack().mostRecentOpToken,
				parser.stack().mostRecentOpToken,
			)
		}
		// We have "p1 -and <unknown_token>/<incomplete_operator>". Pop the binary op off
		// the stack and include it as part of the remaining tokens. This is useful in case
		// our expression is inside another expression, where the top-level expression handles
		// combining our parsed predicate p with whatever's parsed by the
		// "<unknown_token>/<incomplete_operator" bit. For example, it ensures that the top-level
		// `wash find` parser correctly parses something like "-m .key foo -o -m .key bar" as
		// "Meta(.key, foo) -o Meta(.key, bar)".
		parser.stack().Pop()
		tokens = append([]string{parser.stack().mostRecentOpToken}, tokens...)
	}
	// Call s.evaluate() to handle cases like "p1 -and p2"
	parser.stack().evaluate()
	return parser.stack().Pop().(predicate.Predicate), tokens, err
}

type evalStack struct {
	*stack.Stack
	andOp             *BinaryOp
	mostRecentOp      *BinaryOp
	mostRecentOpToken string
}

func newEvalStack(andOp *BinaryOp) *evalStack {
	return &evalStack{
		andOp: andOp,
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
		s.pushBinaryOp(s.andOp.tokens[0], s.andOp)
	}
	s.Push(p)
}

func (s *evalStack) evaluate() {
	// Invariant: s's layout is something like "p (<op> p)*"
	for s.Len() > 1 {
		p2 := s.Pop().(predicate.Predicate)
		op := s.Pop().(*BinaryOp)
		p1 := s.Pop().(predicate.Predicate)
		s.Push(op.op.Combine(p1, p2))
	}
}

type emptyExpressionError struct {
	msg string
}

func (e emptyExpressionError) Error() string {
	return e.msg
}
