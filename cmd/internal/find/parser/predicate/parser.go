package predicate

import "github.com/puppetlabs/wash/cmd/internal/find/parser/errz"

// Parser represents a parser that parses predicates.
type Parser interface {
	Parse(tokens []string) (Predicate, []string, error)
}

// CompositeParser represents a parser composed of multiple predicate
// parsers.
type CompositeParser struct {
	ErrMsg string
	Parsers []Parser
}

// Parse is a wrapper to ParseAndReturnParserID. It implements the predicate.Parser
// interface for a CompositeParser.
func (cp CompositeParser) Parse(tokens []string) (Predicate, []string, error) {
	p, tokens, _, err := cp.ParseAndReturnParserID(tokens)
	return p, tokens, err
}

// ParseAndReturnParserID attempts to parse a predicate from the given tokens. It loops
// through each of cp's parsers, returning the result of the first parser that matches
// the input, and the matching parser's ID. If no parser matches the input, then Parse
// returns a MatchError containing cp.ErrMsg
func (cp CompositeParser) ParseAndReturnParserID(tokens []string) (Predicate, []string, int, error) {
	for i, parser := range cp.Parsers {
		p, tokens, err := parser.Parse(tokens)
		if err == nil {
			return p, tokens, i, nil
		}
		if !errz.IsMatchError(err) {
			// Parser matched the input, but returned a syntax error. Return
			// the error.
			return nil, nil, i, err
		}
	}
	// None of the parsers matched the input, so return a MatchError
	return nil, nil, -1, errz.NewMatchError(cp.ErrMsg)		
}

// ToParser converts the given parse function to a predicate.Parser object
func ToParser(parseFunc func(tokens []string) (Predicate, []string, error)) Parser {
	return predicateParser(parseFunc)
}

type predicateParser func(tokens []string) (Predicate, []string, error)

// Parse parses a predicate from the given tokens
func (p predicateParser) Parse(tokens []string) (Predicate, []string, error) {
	return p(tokens)
}
