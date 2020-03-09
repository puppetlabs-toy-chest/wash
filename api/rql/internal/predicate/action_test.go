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

func (s *ActionTestSuite) TestUnmarshalErrors() {
	s.UMETC(1, `1.*valid.*action.*"exec"`, true)
	s.UMETC("foo", `foo.*valid.*action.*"exec"`, true)
}

func (s *ActionTestSuite) TestEvalAction() {
	s.EAFTC("exec", plugin.ListAction())
	s.EATTC("exec", plugin.ExecAction())
}

func (s *ActionTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("action", true, func() rql.ASTNode {
			return Action(plugin.Action{})
		})
	}

	s.EAFTC("exec", plugin.ListAction())
	s.EATTC("exec", plugin.ExecAction())
	s.AssertNotImplemented(
		"exec",
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
	)

	s.EATTC(s.A("NOT", "exec"), plugin.ListAction())
	s.EAFTC(s.A("NOT", "exec"), plugin.ExecAction())
}

func TestAction(t *testing.T) {
	s := new(ActionTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Action(plugin.Action{})
	}
	suite.Run(t, s)
}
