package primary

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
)

// Get retrieves the specified primary
func Get(name string) *Primary {
	return Parser.primaryMap[tokenize(name)]
}

// IsSet returns true if the specified primary was set
// Call this after `wash find` finishes parsing its
// arguments
func IsSet(p *Primary) bool {
	return Parser.SetPrimaries[p]
}

// Table returns a table containing all of `wash find`'s available primaries
func Table() *cmdutil.Table {
	rows := make([][]string, len(Parser.primaries))
	for i, p := range Parser.primaries {
		rows[i] = make([]string, 2)
		padding := 6
		if p.shortName != "" {
			padding = 2
		}
		rows[i][0] = strings.Repeat(" ", padding) + p.Usage()
		rows[i][1] = p.Description
	}
	// Now include the "Primaries:" header row
	rows = append(
		[][]string{
			[]string{"Primaries:", ""},
		},
		rows...,
	)
	return cmdutil.NewTable(rows...)
}

// Parser parses `wash find` primaries.
var Parser = &parser{
	primaryMap:   make(map[string]*Primary),
	SetPrimaries: make(map[*Primary]bool),
}

type parser struct {
	// SetPrimaries is exported so that the tests can
	// mock it
	SetPrimaries map[*Primary]bool
	primaryMap   map[string]*Primary
	primaries    []*Primary
}

// IsPrimary returns true if the token is a `wash find`
// primary
func (parser *parser) IsPrimary(token string) bool {
	_, ok := parser.primaryMap[token]
	return ok
}

func (parser *parser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a primary")
	}
	token := tokens[0]
	primary, ok := parser.primaryMap[token]
	if !ok {
		msg := fmt.Sprintf("%v: unknown primary", token)
		return nil, nil, errz.NewMatchError(msg)
	}
	tokens = tokens[1:]
	p, tokens, err := primary.Parse(tokens)
	if errz.IsSyntaxError(err) {
		return nil, nil, fmt.Errorf("%v: %v", token, err)
	}
	parser.SetPrimaries[primary] = true
	return p, tokens, err
}

func (parser *parser) add(p *Primary) *Primary {
	p.tokens = make(map[string]struct{})
	parser.primaries = append(parser.primaries, p)
	for _, name := range []string{p.name, p.shortName} {
		if name != "" {
			token := tokenize(name)
			p.tokens[token] = struct{}{}
			parser.primaryMap[token] = p
		}
	}
	return p
}

// Primary represents a `wash find` primary.
type Primary struct {
	Description         string
	DetailedDescription string
	args                string
	shortName           string
	name                string
	tokens              map[string]struct{}
	parseFunc           types.EntryPredicateParser
}

// Usage returns the primary's usage string.
func (primary *Primary) Usage() string {
	nameTk := tokenize(primary.name)
	usage := fmt.Sprintf("%v", nameTk)
	if primary.shortName != "" {
		shortNameTk := tokenize(primary.shortName)
		usage = fmt.Sprintf("%v, %v", shortNameTk, nameTk)
	}
	if primary.args != "" {
		usage += " " + primary.args
	}
	return usage
}

// Parse parses a predicate from the given primary.
func (primary *Primary) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return primary.parseFunc(tokens)
}

func tokenize(primaryName string) string {
	return "-" + primaryName
}
