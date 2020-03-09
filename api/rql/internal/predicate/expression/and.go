package expression

import (
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

func And(p1 rql.ASTNode, p2 rql.ASTNode) rql.ASTNode {
	p := &and{binOp{
		op: "AND",
		p1: toAtom(p1),
		p2: toAtom(p2),
	}}
	p.ValuePredicateBase = meta.NewValuePredicate(p)
	return p
}

type and struct {
	binOp
}

func (a *and) EvalEntry(e rql.Entry) bool {
	ep1 := a.p1.(rql.EntryPredicate)
	ep2 := a.p2.(rql.EntryPredicate)
	return ep1.EvalEntry(e) && ep2.EvalEntry(e)
}

func (a *and) EvalEntrySchema(s *rql.EntrySchema) bool {
	esp1 := a.p1.(rql.EntrySchemaPredicate)
	esp2 := a.p2.(rql.EntrySchemaPredicate)
	return esp1.EvalEntrySchema(s) && esp2.EvalEntrySchema(s)
}

func (a *and) EvalValue(v interface{}) bool {
	vp1 := a.p1.(rql.ValuePredicate)
	vp2 := a.p2.(rql.ValuePredicate)
	return vp1.EvalValue(v) && vp2.EvalValue(v)
}

func (a *and) EvalString(str string) bool {
	sp1 := a.p1.(rql.StringPredicate)
	sp2 := a.p2.(rql.StringPredicate)
	return sp1.EvalString(str) && sp2.EvalString(str)
}

func (a *and) EvalNumeric(x decimal.Decimal) bool {
	np1 := a.p1.(rql.NumericPredicate)
	np2 := a.p2.(rql.NumericPredicate)
	return np1.EvalNumeric(x) && np2.EvalNumeric(x)
}

func (a *and) EvalTime(t time.Time) bool {
	tp1 := a.p1.(rql.TimePredicate)
	tp2 := a.p2.(rql.TimePredicate)
	return tp1.EvalTime(t) && tp2.EvalTime(t)
}

func (a *and) EvalAction(action plugin.Action) bool {
	ap1 := a.p1.(rql.ActionPredicate)
	ap2 := a.p2.(rql.ActionPredicate)
	return ap1.EvalAction(action) && ap2.EvalAction(action)
}

func (a *and) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	sp1 := a.p1.(meta.ValuePredicate).SchemaPredicate(svs)
	sp2 := a.p2.(meta.ValuePredicate).SchemaPredicate(svs)
	return func(schema meta.ValueSchema) bool {
		return sp1(schema) && sp2(schema)
	}
}

var _ = rql.EntryPredicate(&and{})
var _ = rql.EntrySchemaPredicate(&and{})
var _ = meta.ValuePredicate(&and{})
var _ = rql.StringPredicate(&and{})
var _ = rql.NumericPredicate(&and{})
var _ = rql.TimePredicate(&and{})
var _ = rql.ActionPredicate(&and{})
