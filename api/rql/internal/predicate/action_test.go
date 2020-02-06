package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type ActionTestSuite struct {
	asttest.Suite
}

func (s *ActionTestSuite) TestMarshal() {
	s.MTC(Action(plugin.ExecAction()), "exec")
}

func (s *ActionTestSuite) TestUnmarshal() {
	a := Action(plugin.Action{})
	s.UMETC(a, 1, `1.*valid.*action.*"exec"`, true)
	s.UMETC(a, "foo", `foo.*valid.*action.*"exec"`, true)
	// UMTC doesn't work because s.Equal doesn't work for the Action
	// type. My best guess is because the Action type has a function
	// as its field, and s.Equal doesn't work with functions. Thus, we
	// do our own assertion here.
	if s.NoError(a.Unmarshal("exec")) {
		s.True(EqualAction(a, "exec"))
	}
}

func (s *ActionTestSuite) TestEvalAction() {
	a := Action(plugin.ExecAction())
	s.EAFTC(a, plugin.ListAction())
	s.EATTC(a, plugin.ExecAction())
}

func (s *ActionTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("action", func() rql.ASTNode {
		return Action(plugin.Action{})
	})

	s.MUM(expr, "exec")
	s.EAFTC(expr, plugin.ListAction())
	s.EATTC(expr, plugin.ExecAction())
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", "exec"})
	s.EATTC(expr, plugin.ListAction())
	s.EAFTC(expr, plugin.ExecAction())
}

func TestAction(t *testing.T) {
	suite.Run(t, new(ActionTestSuite))
}
