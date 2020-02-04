package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func CName(p rql.StringPredicate) rql.Primary {
	return &cname{
		base: base{
			name:  "cname",
			ptype: "String",
			p:     p,
		},
		p: p,
	}
}

type cname struct {
	base
	p rql.StringPredicate
}

func (p *cname) EvalEntry(e rql.Entry) bool {
	return p.p.EvalString(e.CName)
}

var _ = rql.EntryPredicate(&cname{})
