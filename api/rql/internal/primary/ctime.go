package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Ctime(p rql.TimePredicate) rql.Primary {
	return &ctime{
		base: base{
			name:  "ctime",
			ptype: "Time",
			p:     p,
		},
		p: p,
	}
}

type ctime struct {
	base
	p rql.TimePredicate
}

func (p *ctime) EvalEntry(e rql.Entry) bool {
	return p.p.EvalTime(e.Attributes.Ctime())
}

var _ = rql.EntryPredicate(&ctime{})
