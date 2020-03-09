package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Crtime(p rql.TimePredicate) rql.Primary {
	return &crtime{
		base: base{
			name:  "crtime",
			ptype: "Time",
			p:     p,
		},
		p: p,
	}
}

type crtime struct {
	base
	p rql.TimePredicate
}

func (p *crtime) EvalEntry(e rql.Entry) bool {
	return p.p.EvalTime(e.Attributes.Crtime())
}

var _ = rql.EntryPredicate(&crtime{})
