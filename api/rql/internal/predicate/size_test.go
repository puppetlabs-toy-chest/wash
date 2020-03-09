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

func (s *SizeTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `size.*formatted.*"size".*PE NumericPredicate`, true)
	s.UMETC(s.A("foo"), `size.*formatted.*"size".*PE NumericPredicate`, true)
	s.UMETC(s.A("size", "foo", "bar"), `size.*formatted.*"size".*PE NumericPredicate`, false)
	s.UMETC(s.A("size"), `size.*formatted.*"size".*PE NumericPredicate.*missing.*PE NumericPredicate`, false)
	s.UMETC(s.A("size", s.A("<", true)), "size.*PE NumericPredicate.*valid.*number", false)
	s.UMETC(s.A("size", s.A("<", "-10")), "size.*PE NumericPredicate.*unsigned.*number", false)
}

func (s *SizeTestSuite) TestEvalEntry() {
	ast := s.A("size", s.A(">", "0"))
	e := rql.Entry{}
	e.Attributes.SetSize(uint64(0))
	s.EEFTC(ast, e)
	e.Attributes.SetSize(uint64(1))
	s.EETTC(ast, e)
}

func (s *SizeTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("size", true, func() rql.ASTNode {
			return Size(UnsignedNumeric("", s.N("0")))
		})
	}

	ast := s.A("size", s.A(">", "0"))
	e := rql.Entry{}
	e.Attributes.SetSize(uint64(0))
	s.EEFTC(ast, e)
	e.Attributes.SetSize(uint64(1))
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
	s.EESTTC(ast, schema)

	s.AssertNotImplemented(
		ast,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestSize(t *testing.T) {
	s := new(SizeTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Size(UnsignedNumeric("", s.N("0")))
	}
	suite.Run(t, s)
}
