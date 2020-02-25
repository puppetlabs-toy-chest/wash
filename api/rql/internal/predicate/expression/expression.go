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
The AtomGenerator should generate "empty" structs representing
a 'ptype' predicate.

A PE is described by the following grammar:
	PE := NOT(PE) | AND(PE, PE) | OR(PE, PE) | Atom(ptype)
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
	ptype string
	g     PtypeGenerator
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
	return nil
}

func (expr *expression) EvalEntry(e rql.Entry) bool {
	return expr.MatchedNode().(rql.EntryPredicate).EvalEntry(e)
}

func (expr *expression) EvalEntrySchema(s *rql.EntrySchema) bool {
	return expr.MatchedNode().(rql.EntrySchemaPredicate).EvalEntrySchema(s)
}

func (expr *expression) EvalValue(v interface{}) bool {
	return expr.MatchedNode().(rql.ValuePredicate).EvalValue(v)
}

func (expr *expression) EvalString(str string) bool {
	return expr.MatchedNode().(rql.StringPredicate).EvalString(str)
}

func (expr *expression) EvalNumeric(x decimal.Decimal) bool {
	return expr.MatchedNode().(rql.NumericPredicate).EvalNumeric(x)
}

func (expr *expression) EvalTime(t time.Time) bool {
	return expr.MatchedNode().(rql.TimePredicate).EvalTime(t)
}

func (expr *expression) EvalAction(action plugin.Action) bool {
	return expr.MatchedNode().(rql.ActionPredicate).EvalAction(action)
}

var _ = expressionNode(&expression{})
var _ = rql.EntryPredicate(&expression{})
var _ = rql.EntrySchemaPredicate(&expression{})
var _ = rql.ValuePredicate(&expression{})
var _ = rql.StringPredicate(&expression{})
var _ = rql.NumericPredicate(&expression{})
var _ = rql.TimePredicate(&expression{})
var _ = rql.ActionPredicate(&expression{})
