package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Meta(p rql.ValuePredicate) rql.Primary {
	return &meta{
		base: base{
			name:         "meta",
			ptype:        "Object",
			notNegatable: true,
			p:            p,
		},
		p: p,
	}
}

type meta struct {
	base
	p rql.ValuePredicate
}

func (p *meta) EvalEntry(e rql.Entry) bool {
	return p.p.EvalValue(e.Metadata)
}

func (p *meta) EvalEntrySchema(s *rql.EntrySchema) bool {
	if s.MetadataSchema() == nil {
		// Metadata schemas are hard to generate in dynamic languages
		// like Ruby/Python. Thus, we choose not to require them for
		// a better UX.
		return true
	}
	return p.p.EvalValueSchema(s.MetadataSchema())
}

var _ = rql.EntryPredicate(&meta{})
var _ = rql.EntrySchemaPredicate(&meta{})
