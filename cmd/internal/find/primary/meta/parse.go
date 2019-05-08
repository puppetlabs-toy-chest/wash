// Package meta contains all the parsing logic for the `meta` primary
package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// The functionality here is tested in primary/meta_test.go

// Parse is the meta primary's parse function.
func Parse(tokens []string) (types.EntryPredicate, []string, error) {
	p, tokens, err := parseExpression(tokens)
	if err != nil {
		return nil, nil, err
	}
	return func(e types.Entry) bool {
		mp := map[string]interface{}(e.Attributes.Meta())
		return p.IsSatisfiedBy(mp)
	}, tokens, nil
}
