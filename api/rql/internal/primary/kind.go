package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

// TODO: Remember to munge s.Path() to appropriate Kind values
// in the walker
func Kind(p rql.StringPredicate) rql.Primary {
	return &kind{
		base: base{
			name:  "kind",
			ptype: "string",
			p:     p,
		},
		p: p,
	}
}

type kind struct {
	base
	p rql.StringPredicate
}

func (p *kind) EntrySchemaInDomain(s *rql.EntrySchema) bool {
	// kind makes sense for schemas that aren't the stree root
	return len(s.Path()) > 0
}

func (p *kind) EvalEntrySchema(s *rql.EntrySchema) bool {
	return p.p.EvalString(s.Path())
}

func (p *kind) EntryInDomain(e rql.Entry) bool {
	// kind only makes sense for entries with schemas
	return e.Schema != nil
}

var _ = rql.EntrySchemaPredicate(&kind{})
