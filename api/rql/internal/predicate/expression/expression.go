package expression

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

type PtypeGenerator func() rql.ASTNode

/*
New returns a new predicate expression of 'ptype' predicates (PE).
The AtomGenerator should generate "emtpy" structs representing
a 'ptype' predicate.

A PE is described by the following grammar:
	PE := NOT(PE) | AND(PE, PE) | OR(PE, PE) | Atom(ptype)
When evaluating the PE, we use a reduced version of the expression. The reduced
version ensures that we correctly implement predicates with a *InDomain method
(like EntryPredicates, EntrySchemaPredicates, ValuePredicates) without having to
write a lot of code. See the "reduce" method's implementation for details on how a
given PE's reduced.
*/
func New(ptype string, g PtypeGenerator) rql.ASTNode {
	e := &expression{
		base:  base{},
		ptype: ptype,
		g:     g,
	}
	return e
}

type expression struct {
	base
	internal.NonterminalNode
	ptype       string
	g           PtypeGenerator
	reducedForm expressionNode
}

func (expr *expression) Unmarshal(input interface{}) error {
	expr.NonterminalNode = internal.NewNonterminalNode(
		Not(New(expr.ptype, expr.g)),
		And(New(expr.ptype, expr.g), New(expr.ptype, expr.g)),
		Or(New(expr.ptype, expr.g), New(expr.ptype, expr.g)),
		Atom(expr.g()),
	)
	expr.SetMatchErrMsg(fmt.Sprintf("expected PE %v", expr.ptype))
	if err := expr.NonterminalNode.Unmarshal(input); err != nil {
		if errz.IsMatchError(err) {
			return err
		}
		return fmt.Errorf("failed to unmarshal PE %v: %w", expr.ptype, err)
	}
	expr.reducedForm = reduce(expr.MatchedNode())
	return nil
}

func (expr *expression) EvalEntry(e rql.Entry) bool {
	return expr.reducedForm.(rql.EntryPredicate).EvalEntry(e)
}

func (expr *expression) EvalEntrySchema(s *rql.EntrySchema) bool {
	return expr.reducedForm.(rql.EntrySchemaPredicate).EvalEntrySchema(s)
}

func (expr *expression) EvalValue(v interface{}) bool {
	return expr.reducedForm.(rql.ValuePredicate).EvalValue(v)
}

func (expr *expression) EvalString(str string) bool {
	return expr.reducedForm.(rql.StringPredicate).EvalString(str)
}

func (expr *expression) EvalNumeric(x decimal.Decimal) bool {
	return expr.reducedForm.(rql.NumericPredicate).EvalNumeric(x)
}

func (expr *expression) EvalTime(t time.Time) bool {
	return expr.reducedForm.(rql.TimePredicate).EvalTime(t)
}

func (expr *expression) EvalAction(action plugin.Action) bool {
	return expr.reducedForm.(rql.ActionPredicate).EvalAction(action)
}

/*
reduce reduces the given PE. A reduced predicate expression of 'ptype' predicates
(RPE) has the following grammar:
	RPE := Not(Atom(ptype)) | And(RPE, RPE) | Or(RPE, RPE) | Atom(ptype)
Note that the key difference between a PE and an RPE is that the NOT operator
in an RPE can only be associated with Atoms instead of other RPEs. As an example,
given the following PE
	AND(OR(A1, NOT(A2)), NOT(OR(NOT(AND(A3, A4))), A5))
Its corresponding RPE is
	AND(OR(A1, NOT(A2)), AND(AND(A3, A4), NOT(A5)))
where we used De'Morgan's law to distribute the NOT inside the OR, and noted
that NOT(NOT(p)) == p.
*/
func reduce(exp rql.ASTNode) expressionNode {
	switch t := exp.(type) {
	default:
		panic(fmt.Sprintf("unknown predicate expression node %T", t))
	case *atom:
		return t
	case *and:
		return And(reduce(t.p1), reduce(t.p2)).(expressionNode)
	case *or:
		return Or(reduce(t.p1), reduce(t.p2)).(expressionNode)
	case *not:
		switch p := t.p.(type) {
		default:
			panic(fmt.Sprintf("unknown predicate expression node %T", p))
		case *atom:
			// NOT(p) is already reduced
			return t
		case *not:
			// NOT(NOT(p)) == p
			return reduce(p.p)
		case *and:
			// NOT(AND(p1, p2)) == OR(NOT(p1), NOT(p2))
			return reduce(Or(Not(p.p1), Not(p.p2)))
		case *or:
			// NOT(OR(p1, p2)) == AND(NOT(p1), NOT(p2))
			return reduce(And(Not(p.p1), Not(p.p2)))
		}
	}
}

var _ = expressionNode(&expression{})
var _ = rql.EntryPredicate(&expression{})
var _ = rql.EntrySchemaPredicate(&expression{})
var _ = rql.ValuePredicate(&expression{})
var _ = rql.StringPredicate(&expression{})
var _ = rql.NumericPredicate(&expression{})
var _ = rql.TimePredicate(&expression{})
var _ = rql.ActionPredicate(&expression{})
