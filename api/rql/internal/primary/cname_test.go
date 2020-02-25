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

func (s *CNameTestSuite) TestUnmarshal() {
	n := CName(predicate.StringGlob(""))
	s.UMETC(n, "foo", `cname.*formatted.*"cname".*PE StringPredicate`, true)
	s.UMETC(n, s.A("foo", s.A("glob", "foo")), `cname.*formatted.*"cname".*PE StringPredicate`, true)
	s.UMETC(n, s.A("cname", "foo", "bar"), `cname.*formatted.*"cname".*PE StringPredicate`, false)
	s.UMETC(n, s.A("cname"), `cname.*formatted.*"cname".*PE StringPredicate.*missing.*PE StringPredicate`, false)
	s.UMETC(n, s.A("cname", s.A("glob", "[")), "cname.*PE StringPredicate.*glob", false)
	s.UMTC(n, s.A("cname", s.A("glob", "foo")), CName(predicate.StringGlob("foo")))
}

func (s *CNameTestSuite) TestEvalEntry() {
	n := CName(predicate.StringGlob("foo"))
	e := rql.Entry{}
	e.CName = "bar"
	s.EEFTC(n, e)
	e.CName = "foo"
	s.EETTC(n, e)
}

func (s *CNameTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("cname", func() rql.ASTNode {
		return CName(predicate.String())
	})

	s.MUM(expr, []interface{}{"cname", []interface{}{"glob", "foo"}})
	e := rql.Entry{}
	e.CName = "bar"
	s.EEFTC(expr, e)
	e.CName = "foo"
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"cname", []interface{}{"glob", "foo"}}})
	e.CName = "bar"
	s.EETTC(expr, e)
	e.CName = "foo"
	s.EEFTC(expr, e)

	//s.EESTTC(expr, schema)
}

func TestCName(t *testing.T) {
	suite.Run(t, new(CNameTestSuite))
}
