package expression

import (
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

func Or(p1 rql.ASTNode, p2 rql.ASTNode) rql.ASTNode {
	p := &or{binOp{
		op: "OR",
		p1: toAtom(p1),
		p2: toAtom(p2),
	}}
	p.ValuePredicateBase = meta.NewValuePredicate(p)
	return p
}

type or struct {
	binOp
}

func (o *or) EvalEntry(e rql.Entry) bool {
	ep1 := o.p1.(rql.EntryPredicate)
	ep2 := o.p2.(rql.EntryPredicate)
	return ep1.EvalEntry(e) || ep2.EvalEntry(e)
}

func (o *or) EvalEntrySchema(s *rql.EntrySchema) bool {
	esp1 := o.p1.(rql.EntrySchemaPredicate)
	esp2 := o.p2.(rql.EntrySchemaPredicate)
	return esp1.EvalEntrySchema(s) || esp2.EvalEntrySchema(s)
}

func (o *or) EvalValue(v interface{}) bool {
	vp1 := o.p1.(rql.ValuePredicate)
	vp2 := o.p2.(rql.ValuePredicate)
	return vp1.EvalValue(v) || vp2.EvalValue(v)
}

func (o *or) EvalString(str string) bool {
	sp1 := o.p1.(rql.StringPredicate)
	sp2 := o.p2.(rql.StringPredicate)
	return sp1.EvalString(str) || sp2.EvalString(str)
}

func (o *or) EvalNumeric(x decimal.Decimal) bool {
	np1 := o.p1.(rql.NumericPredicate)
	np2 := o.p2.(rql.NumericPredicate)
	return np1.EvalNumeric(x) || np2.EvalNumeric(x)
}

func (o *or) EvalTime(t time.Time) bool {
	tp1 := o.p1.(rql.TimePredicate)
	tp2 := o.p2.(rql.TimePredicate)
	return tp1.EvalTime(t) || tp2.EvalTime(t)
}

func (o *or) EvalAction(action plugin.Action) bool {
	ap1 := o.p1.(rql.ActionPredicate)
	ap2 := o.p2.(rql.ActionPredicate)
	return ap1.EvalAction(action) || ap2.EvalAction(action)
}

func (o *or) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	sp1 := o.p1.(meta.ValuePredicate).SchemaPredicate(svs)
	sp2 := o.p2.(meta.ValuePredicate).SchemaPredicate(svs)
	return func(schema meta.ValueSchema) bool {
		return sp1(schema) || sp2(schema)
	}
}

var _ = expressionNode(&or{})
var _ = rql.EntryPredicate(&or{})
var _ = rql.EntrySchemaPredicate(&or{})
var _ = meta.ValuePredicate(&or{})
var _ = rql.StringPredicate(&or{})
var _ = rql.NumericPredicate(&or{})
var _ = rql.TimePredicate(&or{})
var _ = rql.ActionPredicate(&or{})
