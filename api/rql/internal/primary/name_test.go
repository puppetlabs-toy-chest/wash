package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type NameTestSuite struct {
	asttest.Suite
}

func (s *NameTestSuite) TestMarshal() {
	s.MTC(Name(predicate.StringGlob("foo")), s.A("name", s.A("glob", "foo")))
}

func (s *NameTestSuite) TestUnmarshal() {
	s.UMETC("foo", `name.*formatted.*"name".*NPE StringPredicate`, true)
	s.UMETC(s.A("foo", s.A("glob", "foo")), `name.*formatted.*"name".*NPE StringPredicate`, true)
	s.UMETC(s.A("name", "foo", "bar"), `name.*formatted.*"name".*NPE StringPredicate`, false)
	s.UMETC(s.A("name"), `name.*formatted.*"name".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(s.A("name", s.A("glob", "[")), "name.*NPE StringPredicate.*glob", false)
}

func (s *NameTestSuite) TestEvalEntry() {
	ast := s.A("name", s.A("glob", "foo"))
	e := rql.Entry{}
	e.Name = "bar"
	s.EEFTC(ast, e)
	e.Name = "foo"
	s.EETTC(ast, e)
}

func (s *NameTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("name", false, func() rql.ASTNode {
			return Name(predicate.String())
		})
	}

	ast := s.A("name", s.A("glob", "foo"))
	e := rql.Entry{}
	e.Name = "bar"
	s.EEFTC(ast, e)
	e.Name = "foo"
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
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

func TestName(t *testing.T) {
	s := new(NameTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Name(predicate.String())
	}
	suite.Run(t, s)
}
