package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Name(p rql.StringPredicate) rql.Primary {
	return &name{
		base: base{
			name:  "name",
			ptype: "String",
			p:     p,
		},
		p: p,
	}
}

type name struct {
	base
	p rql.StringPredicate
}

func (p *name) EvalEntry(e rql.Entry) bool {
	return p.p.EvalString(e.Name)
}

var _ = rql.EntryPredicate(&name{})
