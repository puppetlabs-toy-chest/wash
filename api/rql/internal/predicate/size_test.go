package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

// EvalValue is tested in the respective tests for object/array predicates

type SizeTestSuite struct {
	asttest.Suite
}

func (s *SizeTestSuite) TestMarshal() {
	s.MTC(Size(UnsignedNumeric(LT, s.N("10"))), s.A("size", s.A("<", "10")))
}

func (s *SizeTestSuite) TestUnmarshal() {
	p := Size(UnsignedNumeric("", s.N("0")))
	s.UMETC(p, "foo", `size.*formatted.*"size".*PE NumericPredicate`, true)
	s.UMETC(p, s.A("foo"), `size.*formatted.*"size".*PE NumericPredicate`, true)
	s.UMETC(p, s.A("size", "foo", "bar"), `size.*formatted.*"size".*PE NumericPredicate`, false)
	s.UMETC(p, s.A("size"), `size.*formatted.*"size".*PE NumericPredicate.*missing.*PE NumericPredicate`, false)
	s.UMETC(p, s.A("size", s.A("<", true)), "size.*PE NumericPredicate.*valid.*number", false)
	s.UMETC(p, s.A("size", s.A("<", "-10")), "size.*PE NumericPredicate.*unsigned.*number", false)
	s.UMTC(p, s.A("size", s.A("<", "10")), Size(UnsignedNumeric(LT, s.N("10"))))
}

func (s *SizeTestSuite) TestEvalEntry() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	e := rql.Entry{}
	e.Attributes.SetSize(uint64(0))
	s.EEFTC(p, e)
	e.Attributes.SetSize(uint64(1))
	s.EETTC(p, e)
}

func (s *SizeTestSuite) TestExpression_Atom() {
	expr := expression.New("size", true, func() rql.ASTNode {
		return Size(UnsignedNumeric("", s.N("0")))
	})

	s.MUM(expr, []interface{}{"size", []interface{}{">", "0"}})

	e := rql.Entry{}
	e.Attributes.SetSize(uint64(0))
	s.EEFTC(expr, e)
	e.Attributes.SetSize(uint64(1))
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestSize(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}
