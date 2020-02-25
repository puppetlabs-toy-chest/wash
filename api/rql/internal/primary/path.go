package primary

import (
	"github.com/puppetlabs/wash/api/rql"
)

// TODO: Remember to munge start path and paths in walker
// (symmetry w/ kind)
func Path(p rql.StringPredicate) rql.Primary {
	return &path{
		base: base{
			name:  "path",
			ptype: "String",
			p:     p,
		},
		p: p,
	}
}

type path struct {
	base
	p rql.StringPredicate
}

func (p *path) EvalEntry(e rql.Entry) bool {
	return p.p.EvalString(e.Path)
}

var _ = rql.EntryPredicate(&path{})
