package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type CNameTestSuite struct {
	asttest.Suite
}

func (s *CNameTestSuite) TestMarshal() {
	s.MTC(CName(predicate.StringGlob("foo")), s.A("cname", s.A("glob", "foo")))
}

func (s *CNameTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `cname.*formatted.*"cname".*NPE StringPredicate`, true)
	s.UMETC(s.A("foo", s.A("glob", "foo")), `cname.*formatted.*"cname".*NPE StringPredicate`, true)
	s.UMETC(s.A("cname", "foo", "bar"), `cname.*formatted.*"cname".*NPE StringPredicate`, false)
	s.UMETC(s.A("cname"), `cname.*formatted.*"cname".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(s.A("cname", s.A("glob", "[")), "cname.*NPE StringPredicate.*glob", false)
}

func (s *CNameTestSuite) TestEvalEntry() {
	ast := s.A("cname", s.A("glob", "foo"))
	e := rql.Entry{}
	e.CName = "bar"
	s.EEFTC(ast, e)
	e.CName = "foo"
	s.EETTC(ast, e)
}

func (s *CNameTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("cname", false, func() rql.ASTNode {
			return CName(predicate.String())
		})
	}

	ast := s.A("cname", s.A("glob", "foo"))
	e := rql.Entry{}
	e.CName = "bar"
	s.EEFTC(ast, e)
	e.CName = "foo"
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

func TestCName(t *testing.T) {
	s := new(CNameTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return CName(predicate.String())
	}
	suite.Run(t, s)
}
