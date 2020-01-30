package predicate

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/plugin"
)

func Action(a plugin.Action) rql.ActionPredicate {
	return &action{
		a: a,
	}
}

type action struct {
	a plugin.Action
}

func (p *action) Marshal() interface{} {
	return p.a.Name
}

func (p *action) Unmarshal(input interface{}) error {
	name, ok := input.(string)
	if !ok {
		return errz.MatchErrorf("must be formatted as <action>")
	}
	a, ok := plugin.Actions()[name]
	if !ok {
		return errz.MatchErrorf("must be formatted as <action>")
	}
	p.a = a
	return nil
}

func (p *action) EvalAction(action plugin.Action) bool {
	return p.a.Name == action.Name
}

// This is for the tests
func EqualAction(p rql.ASTNode, a string) bool {
	return p.(*action).a.Name == a
}

var _ = rql.ActionPredicate(&action{})
