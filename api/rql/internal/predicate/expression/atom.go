package expression

import (
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

func Atom(p rql.ASTNode) rql.ASTNode {
	if _, ok := p.(expressionNode); ok {
		panic("expression.Atom was called with an expression node")
	}
	return &atom{
		base: base{},
		p:    p,
	}
}

func toAtom(p rql.ASTNode) rql.ASTNode {
	if _, ok := p.(expressionNode); ok {
		return p
	}
	return Atom(p)
}

type atom struct {
	base
	p rql.ASTNode
}

func (a *atom) Marshal() interface{} {
	return a.p.Marshal()
}

func (a *atom) Unmarshal(input interface{}) error {
	if err := a.p.Unmarshal(input); err != nil {
		return err
	}
	a.p = unravelNTN(a.p)
	return nil
}

func (a *atom) EvalEntry(e rql.Entry) bool {
	result := a.p.(rql.Primary).EntryInDomain(e)
	if ep, ok := a.p.(rql.EntryPredicate); ok {
		result = result && ep.EvalEntry(e)
	}
	return result
}

func (a *atom) EvalEntrySchema(s *rql.EntrySchema) bool {
	result := a.p.(rql.Primary).EntrySchemaInDomain(s)
	if sp, ok := a.p.(rql.EntrySchemaPredicate); ok {
		result = result && sp.EvalEntrySchema(s)
	}
	return result
}

func (a *atom) EvalValue(v interface{}) bool {
	vp := a.p.(rql.ValuePredicate)
	return vp.ValueInDomain(v) && vp.EvalValue(v)
}

func (a *atom) EvalString(str string) bool {
	return a.p.(rql.StringPredicate).EvalString(str)
}

func (a *atom) EvalNumeric(x decimal.Decimal) bool {
	return a.p.(rql.NumericPredicate).EvalNumeric(x)
}

func (a *atom) EvalTime(t time.Time) bool {
	return a.p.(rql.TimePredicate).EvalTime(t)
}

func (a *atom) EvalAction(action plugin.Action) bool {
	return a.p.(rql.ActionPredicate).EvalAction(action)
}

var _ = expressionNode(&atom{})
var _ = rql.EntryPredicate(&atom{})
var _ = rql.EntrySchemaPredicate(&atom{})
var _ = rql.ValuePredicate(&atom{})
var _ = rql.StringPredicate(&atom{})
var _ = rql.NumericPredicate(&atom{})
var _ = rql.TimePredicate(&atom{})
var _ = rql.ActionPredicate(&atom{})
