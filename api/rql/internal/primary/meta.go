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

// TODO: Implement EvalEntrySchema

var _ = rql.EntryPredicate(&meta{})
