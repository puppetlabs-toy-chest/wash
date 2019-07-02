// Package meta contains all the parsing logic for the `meta` primary
package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// The functionality here is tested in primary/meta_test.go

// Parse is the meta primary's parse function.
func Parse(tokens []string) (types.EntryPredicate, []string, error) {
	p, tokens, err := parseExpression(tokens)
	if err != nil {
		return nil, nil, err
	}
	entryP := types.ToEntryP(func(e types.Entry) bool {
		return p.IsSatisfiedBy(e.Metadata)
	})
	entryP.SetSchemaP(&entrySchemaPredicate{
		p: p.(Predicate).schemaP(),
	})
	return entryP, tokens, nil
}

// entrySchemaPredicate is the meta primary's entry schema predicate.
type entrySchemaPredicate struct {
	p schemaPredicate
}

func (p *entrySchemaPredicate) IsSatisfiedBy(v interface{}) bool {
	s, ok := v.(*types.EntrySchema)
	if !ok {
		return false
	}
	return p.P(s)
}

func (p *entrySchemaPredicate) Negate() predicate.Predicate {
	return &entrySchemaPredicate{
		p: p.p.Negate().(schemaPredicate),
	}
}

func (p *entrySchemaPredicate) P(s *types.EntrySchema) bool {
	if s.MetadataSchemaPValue == nil {
		// Metadata schemas are hard to generate in dynamic languages
		// like Ruby/Python. Thus, we choose not to require them for
		// a better UX.
		return true
	}
	return p.p.IsSatisfiedBy(newSchema(s.MetadataSchemaPValue))
}
