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

func (s *ActionTestSuite) TestUnmarshal() {
	p := Action(predicate.Action(plugin.Action{}))
	s.UMETC(p, "foo", `action.*formatted.*"action".*PE ActionPredicate`, true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), `action.*formatted.*"action".*PE ActionPredicate`, true)
	s.UMETC(p, s.A("action", "foo", "bar"), `action.*formatted.*"action".*PE ActionPredicate`, false)
	s.UMETC(p, s.A("action"), `action.*formatted.*"action".*PE ActionPredicate.*missing.*PE ActionPredicate`, false)
	s.UMETC(p, s.A("action", "foo"), "<action>", false)
	// UMTC doesn't work because s.Equal doesn't work for the Action
	// type so we do our own assertion here.
	if s.NoError(p.Unmarshal(s.A("action", "exec"))) {
		predicate.EqualAction(p.(*action).p, "exec")
	}
}

func (s *ActionTestSuite) TestEntryInDomain() {
	p := Action(predicate.Action(plugin.ExecAction()))
	s.EIDTTC(p, rql.Entry{})
}

func (s *ActionTestSuite) TestEvalEntry() {
	p := Action(predicate.Action(plugin.ExecAction()))
	e := rql.Entry{}
	e.Actions = []string{"list", "read"}
	s.EEFTC(p, e)
	e.Actions = []string{"list", "exec", "signal"}
	s.EETTC(p, e)
}

func (s *ActionTestSuite) TestEntrySchemaInDomain() {
	p := Action(predicate.Action(plugin.ExecAction()))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *ActionTestSuite) TestEvalEntrySchema() {
	p := Action(predicate.Action(plugin.ExecAction()))
	schema := &rql.EntrySchema{}
	schema.SetActions([]string{"list", "read"})
	s.EESFTC(p, schema)
	schema.SetActions([]string{"list", "exec", "signal"})
	s.EESTTC(p, schema)
}

func (s *ActionTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("action", func() rql.ASTNode {
		return Action(predicate.Action(plugin.Action{}))
	})

	s.MUM(expr, []interface{}{"action", "exec"})
	e := rql.Entry{}
	e.Actions = []string{"list", "read"}
	s.EEFTC(expr, e)
	e.Actions = []string{"list", "exec", "signal"}
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	schema.SetActions([]string{"list", "read"})
	s.EESFTC(expr, schema)
	schema.SetActions([]string{"list", "exec", "signal"})
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"action", "exec"}})
	e.Actions = []string{"list", "read"}
	s.EETTC(expr, e)
	e.Actions = []string{"list", "exec", "signal"}
	s.EEFTC(expr, e)

	schema.SetActions([]string{"list", "read"})
	s.EESTTC(expr, schema)
	schema.SetActions([]string{"list", "exec", "signal"})
	s.EESFTC(expr, schema)
}

func TestAction(t *testing.T) {
	suite.Run(t, new(ActionTestSuite))
}
