package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Atime(p rql.TimePredicate) rql.Primary {
	return &atime{
		base: base{
			name:  "atime",
			ptype: "Time",
			p:     p,
		},
		p: p,
	}
}

type atime struct {
	base
	p rql.TimePredicate
}

func (p *atime) EvalEntry(e rql.Entry) bool {
	return p.p.EvalTime(e.Attributes.Atime())
}

var _ = rql.EntryPredicate(&atime{})
