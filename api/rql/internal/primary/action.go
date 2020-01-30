package primary

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/plugin"
)

func Action(p rql.ActionPredicate) rql.Primary {
	return &action{
		base: base{
			name:  "action",
			ptype: "action",
			p:     p,
		},
		p: p,
	}
}

type action struct {
	base
	p rql.ActionPredicate
}

func (p *action) EvalEntry(e rql.Entry) bool {
	return p.evalActions(e.Actions)
}

func (p *action) EvalEntrySchema(s *rql.EntrySchema) bool {
	return p.evalActions(s.Actions())
}

func (p *action) evalActions(actions []string) bool {
	for _, action := range actions {
		if p.p.EvalAction(plugin.Actions()[action]) {
			return true
		}
	}
	return false
}

var _ = rql.EntryPredicate(&action{})
var _ = rql.EntrySchemaPredicate(&action{})
