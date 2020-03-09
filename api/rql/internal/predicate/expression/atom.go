package expression

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

/*
Atom wraps p into a type that works with predicate expressions. The RQL
will use this to call p's appropriate Eval* methods, where each Eval*
method implements its interface-specific semantics. For example, if p is
a Primary and EntryPredicate, then the returned atom's EvalEntry method
will be evaluated as a.EvalEntry(e) == p.EntryInDomain(e) && p.EvalEntry(e).
Front-end interfaces to the RQL should always use the Atom type when testing
their parsed predicates to ensure correct evaluation semantics.

If you'd like to see where Atoms are being used, then check out expression.go.
*/
func Atom(p rql.ASTNode) rql.ASTNode {
	if _, ok := p.(expressionNode); ok {
		panic("expression.Atom was called with an expression node")
	}
	a := &atom{
		base: base{},
		p:    p,
	}
	a.ValuePredicateBase = meta.NewValuePredicate(a)
	return a
}

func toAtom(p rql.ASTNode) rql.ASTNode {
	if _, ok := p.(expressionNode); ok {
		return p
	}
	return Atom(p)
}

type atom struct {
	base
	*meta.ValuePredicateBase
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
	_, ok := a.p.(rql.Primary)
	if !ok {
		panic(fmt.Sprintf("Atom#EvalEntry: predicate %T doesn't implement rql.Primary", a.p))
	}
	result := true
	if ep, ok := a.p.(rql.EntryPredicate); ok {
		result = result && ep.EvalEntry(e)
	}
	return result
}

func (a *atom) EvalEntrySchema(s *rql.EntrySchema) bool {
	_, ok := a.p.(rql.Primary)
	if !ok {
		panic(fmt.Sprintf("Atom#EvalEntrySchema: predicate %T doesn't implement rql.Primary", a.p))
	}
	result := true
	if sp, ok := a.p.(rql.EntrySchemaPredicate); ok {
		result = result && sp.EvalEntrySchema(s)
	}
	return result
}

func (a *atom) EvalValue(v interface{}) bool {
	return a.p.(rql.ValuePredicate).EvalValue(v)
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

func (a *atom) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	return a.p.(meta.ValuePredicate).SchemaPredicate(svs)
}

var _ = expressionNode(&atom{})
var _ = rql.EntryPredicate(&atom{})
var _ = rql.EntrySchemaPredicate(&atom{})
var _ = meta.ValuePredicate(&atom{})
var _ = rql.StringPredicate(&atom{})
var _ = rql.NumericPredicate(&atom{})
var _ = rql.TimePredicate(&atom{})
var _ = rql.ActionPredicate(&atom{})
