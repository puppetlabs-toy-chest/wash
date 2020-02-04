package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

func Mtime(p rql.TimePredicate) rql.Primary {
	return &mtime{
		base: base{
			name:  "mtime",
			ptype: "Time",
			p:     p,
		},
		p: p,
	}
}

type mtime struct {
	base
	p rql.TimePredicate
}

func (p *mtime) EvalEntry(e rql.Entry) bool {
	return p.p.EvalTime(e.Attributes.Mtime())
}

var _ = rql.ASTNode(&mtime{})
var _ = rql.EntryPredicate(&mtime{})
