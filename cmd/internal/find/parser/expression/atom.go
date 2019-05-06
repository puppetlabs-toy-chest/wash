package expression

import "github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
import "github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
import "fmt"

func notOpParser(parser *Parser) predicate.Parser {
	return predicate.ToParser(func(tokens []string) (predicate.Predicate, []string, error) {
		notToken := tokens[0]
		if notToken != "!" && notToken != "-not" {
			return nil, nil, errz.NewMatchError("expected ! or -not")
		}
		tokens = tokens[1:]
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("%v: no following expression", notToken)
		}
		if tokens[0] == ")" {
			if parser.numOpenParens <= 0 {
				return nil, nil, fmt.Errorf("): no beginning '('")
			}
			return nil, nil, fmt.Errorf("%v: no following expression", notToken)
		}
		p, tokens, err := parser.atom.Parse(tokens)
		if err != nil {
			if errz.IsMatchError(err) {
				err = fmt.Errorf("%v: no following expression", notToken)
			}
			return nil, nil, err
		}
		return p.Negate(), tokens, err
	})
}

func parensOpParser(parser *Parser) predicate.Parser {
	return predicate.ToParser(func(tokens []string) (predicate.Predicate, []string, error) {
		if tokens[0] != "(" {
			return nil, nil, errz.NewMatchError("expected an '('")
		}
		tokens = tokens[1:]
		parser.numOpenParens++
		// Save the current evaluation stack. We will restore it after parsing
		// the parenthesized expression
		stack := parser.stack
		defer func() {
			parser.stack = stack
			parser.numOpenParens--
		}()
		return parser.Parse(tokens)
	})
}