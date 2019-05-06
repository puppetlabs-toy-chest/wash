package primary

import (
	"fmt"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"strings"
)

// Parser parses `wash find` primaries.
var Parser = &parser{
	CompositeParser: &predicate.CompositeParser{
		ErrMsg: "unknown primary",
	},
	primaries: make(map[string]*primary),
}

type parser struct {
	*predicate.CompositeParser
	primaries map[string]*primary
}

// IsPrimary returns true if the token is a `wash find`
// primary
func (parser *parser) IsPrimary(token string) bool {
	_, ok := parser.primaries[token]
	return ok
}

func (parser *parser) newPrimary(tokens []string, parse func(tokens []string) (predicate.Entry, []string, error)) *primary {
	p := &primary{
		tokens: tokens,
		parseFunc: parse,
	}
	p.tokensMap = make(map[string]struct{})
	for _, token := range tokens {
		p.tokensMap[token] = struct{}{}
		parser.primaries[token] = p
		parser.Parsers = append(parser.Parsers, p)
	}
	return p
}

// primary represents a `wash find` primary.
type primary struct {
	tokens []string
	tokensMap map[string]struct{}
	parseFunc predicate.EntryParser
}

func (primary *primary) parse(tokens []string) (predicate.Entry, []string, error) {
	tokensErrMsg := fmt.Sprintf("expected one of: %v", strings.Join(primary.tokens, ","))
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError(tokensErrMsg)
	}
	token := tokens[0]
	if _, ok := primary.tokensMap[token]; !ok {
		return nil, nil, errz.NewMatchError(tokensErrMsg)
	}
	tokens = tokens[1:]
	p, tokens, err := primary.parseFunc(tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: %v", token, err)
	}
	return p, tokens, nil
}

// Parse parses a predicate from the given primary.
func (primary *primary) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return primary.parse(tokens)
}