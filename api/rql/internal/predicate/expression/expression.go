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
New returns a new predicate expression (PE) of 'ptype' predicates if
negatable is false, otherwise it returns a new negatable predicate
expression (NPE) of 'ptype' predicates.

The AtomGenerator should generate "empty" structs representing
a 'ptype' predicate.

A PE is described by the following grammar:
	PE := AND(PE, PE) | OR(PE, PE) | Atom(ptype)

An NPE is described by the following grammar:
   NPE := NOT(NPE) | AND(NPE, NPE) | OR(NPE, NPE) | Atom(ptype)
*/
func New(ptype string, negatable bool, g PtypeGenerator) rql.ASTNode {
	e := &expression{
		base:      base{},
		ptype:     ptype,
		g:         g,
		negatable: negatable,
	}
	return e
}

type expression struct {
	base
	internal.NonterminalNode
	ptype     string
	g         PtypeGenerator
	negatable bool
}

func (expr *expression) Unmarshal(input interface{}) error {
	exprType := "PE"
	nodes := []rql.ASTNode{
		And(New(expr.ptype, expr.negatable, expr.g), New(expr.ptype, expr.negatable, expr.g)),
		Or(New(expr.ptype, expr.negatable, expr.g), New(expr.ptype, expr.negatable, expr.g)),
		Atom(expr.g()),
	}
	if expr.negatable {
		exprType = "NPE"
		nodes = append(nodes, Not(New(expr.ptype, expr.negatable, expr.g)))
	}
	expr.NonterminalNode = internal.NewNonterminalNode(nodes[0], nodes[1:]...)
	expr.SetMatchErrMsg(fmt.Sprintf("expected %v %v", exprType, expr.ptype))
	if err := expr.NonterminalNode.Unmarshal(input); err != nil {
		if errz.IsMatchError(err) {
			return err
		}
		return fmt.Errorf("failed to unmarshal %v %v: %w", exprType, expr.ptype, err)
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
