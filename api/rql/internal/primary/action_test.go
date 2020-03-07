package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type ActionTestSuite struct {
	asttest.Suite
}

func (s *ActionTestSuite) TestMarshal() {
	s.MTC(Action(predicate.Action(plugin.ExecAction())), s.A("action", "exec"))
}

func (s *ActionTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `action.*formatted.*"action".*NPE ActionPredicate`, true)
	s.UMETC(s.A("foo", s.A("<", int64(1000))), `action.*formatted.*"action".*NPE ActionPredicate`, true)
	s.UMETC(s.A("action", "foo", "bar"), `action.*formatted.*"action".*NPE ActionPredicate`, false)
	s.UMETC(s.A("action"), `action.*formatted.*"action".*PE ActionPredicate.*missing.*NPE ActionPredicate`, false)
	s.UMETC(s.A("action", "foo"), "action.*NPE ActionPredicate.*action", false)
}

func (s *ActionTestSuite) TestEvalEntry() {
	ast := s.A("action", "exec")
	e := rql.Entry{}
	e.Actions = []string{"list", "read"}
	s.EEFTC(ast, e)
	e.Actions = []string{"list", "exec", "signal"}
	s.EETTC(ast, e)
}

func (s *ActionTestSuite) TestEvalEntrySchema() {
	ast := s.A("action", "exec")
	schema := &rql.EntrySchema{}
	schema.SetActions([]string{"list", "read"})
	s.EESFTC(ast, schema)
	schema.SetActions([]string{"list", "exec", "signal"})
	s.EESTTC(ast, schema)
}

func (s *ActionTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("action", false, func() rql.ASTNode {
			return Action(predicate.Action(plugin.Action{}))
		})
	}

	ast := s.A("action", "exec")
	e := rql.Entry{}
	e.Actions = []string{"list", "read"}
	s.EEFTC(ast, e)
	e.Actions = []string{"list", "exec", "signal"}
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
	schema.SetActions([]string{"list", "read"})
	s.EESFTC(ast, schema)
	schema.SetActions([]string{"list", "exec", "signal"})
	s.EESTTC(ast, schema)

	s.AssertNotImplemented(
		ast,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestAction(t *testing.T) {
	s := new(ActionTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Action(predicate.Action(plugin.Action{}))
	}
	suite.Run(t, s)
}
