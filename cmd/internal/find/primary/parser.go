package primary

import (
	"fmt"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
)

// Parser parses `wash find` primaries.
var Parser = &parser{
	primaries: make(map[string]*primary),
}

type parser struct {
	primaries map[string]*primary
}

// IsPrimary returns true if the token is a `wash find`
// primary
func (parser *parser) IsPrimary(token string) bool {
	_, ok := parser.primaries[token]
	return ok
}

func (parser *parser) Parse(tokens []string) (predicate.Predicate, []string, error)  {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a primary")
	}
	token := tokens[0]
	primary, ok := parser.primaries[token]
	if !ok {
		msg := fmt.Sprintf("%v: unknown primary", token)
		return nil, nil, errz.NewMatchError(msg)
	}
	tokens = tokens[1:]
	p, tokens, err := primary.Parse(tokens)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: %v", token, err)
	}
	return p, tokens, nil
}

func (parser *parser) add(p *primary) *primary {
	p.tokensMap = make(map[string]struct{})
	for _, token := range p.tokens {
		p.tokensMap[token] = struct{}{}
		parser.primaries[token] = p
	}
	return p
}

// primary represents a `wash find` primary.
type primary struct {
	tokens []string
	tokensMap map[string]struct{}
	parseFunc types.EntryPredicateParser
}

// Parse parses a predicate from the given primary.
func (primary *primary) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return primary.parseFunc(tokens)
}