package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type NumericTestSuite struct {
	asttest.Suite
}

func (s *NumericTestSuite) TestMarshal() {
	s.MTC(Numeric(LT, s.N("2.3")), s.A("<", "2.3"))
}

func (s *NumericTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", "formatted.*<comparison_op>.*<number>", true)
	s.UMETC(s.A(), "formatted.*<comparison_op>.*<number>", true)
	s.UMETC(s.A("<", "foo", "bar"), "formatted.*<comparison_op>.*<number>", false)
	s.UMETC(s.A("<"), "formatted.*<comparison_op>.*<number>.*missing.*number", false)
	s.UMETC(s.A("<", true), "valid.*number", false)
	s.UMETC(s.A("<", "true"), "parse.*true.*number.*exponent", false)
}

func (s *NumericTestSuite) TestUnmarshalErrors_Unsigned() {
	s.NodeConstructor = func() rql.ASTNode {
		return UnsignedNumeric("", s.N("0"))
	}
	s.UMETC(s.A("<", "-10"), "unsigned.*number", false)
}

func (s *NumericTestSuite) TestEvalNumeric() {
	// Test LT, and also test
	ast := s.A("<", "1")
	s.ENFTC(ast, s.N("2"), s.N("1"))
	s.ENTTC(ast, s.N("0"))

	// Test LTE
	ast = s.A("<=", "1")
	s.ENFTC(ast, s.N("2"))
	s.ENTTC(ast, s.N("0"), s.N("1"))

	// Test GT
	ast = s.A(">", "1")
	s.ENFTC(ast, s.N("0"), s.N("1"))
	s.ENTTC(ast, s.N("2"))

	// Test GTE
	ast = s.A(">=", "1")
	s.ENFTC(ast, s.N("0"))
	s.ENTTC(ast, s.N("2"), s.N("1"))

	// Test EQL
	ast = s.A("=", "1")
	s.ENFTC(ast, s.N("0"), s.N("2"))
	s.ENTTC(ast, s.N("1"))

	// TEST NEQL
	ast = s.A("!=", "1")
	s.ENFTC(ast, s.N("1"))
	s.ENTTC(ast, s.N("0"), s.N("2"))

	// Test that we can unmarshal numbers from different types
	// (float64) and that we can unmarshal very large values
	s.ENTTC(s.A("<", 1), s.N("0"))
	largeV := "10000000000000000000000000000000000000000000000000000000000000000000"
	s.ENTTC(s.A("<", largeV), s.N(largeV[0:10]))
	s.ENTTC(s.A(">", largeV), s.N(largeV+"0"))
}

func (s *NumericTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("numeric", true, func() rql.ASTNode {
			return Numeric("", s.N("0"))
		})
	}

	ast := s.A("<", "1")
	s.ENFTC(ast, s.N("1"))
	s.ENTTC(ast, s.N("0"))
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.ENTTC(notAST, s.N("1"))
	s.ENFTC(notAST, s.N("0"))
}

func TestNumeric(t *testing.T) {
	s := new(NumericTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Numeric("", s.N("0"))
	}
	suite.Run(t, s)
}

type NumericValueTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *NumericValueTestSuite) TestMarshal() {
	s.MTC(NumericValue(Numeric(LT, s.N("2.3"))), s.A("number", s.A("<", "2.3")))
}

func (s *NumericValueTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", "formatted.*number.*NPE NumericPredicate", true)
	s.UMETC(s.A("number", "foo", "bar"), "formatted.*number.*NPE NumericPredicate", false)
	s.UMETC(s.A("number"), "formatted.*number.*NPE NumericPredicate.*missing.*NPE NumericPredicate", false)
	s.UMETC(s.A("number", s.A()), "unmarshalling.*NPE NumericPredicate.*formatted.*<comparison_op>.*<number>", false)
}

func (s *NumericValueTestSuite) TestEvalValue() {
	ast := s.A("number", s.A("<", "2.0"))
	s.EVFTC(ast, 3, "1")
	s.EVTTC(ast, 1)
	// TestEvalNumeric contained the operator-specific test-cases
}

func (s NumericValueTestSuite) TestEvalValueSchema() {
	ast := s.A("number", s.A("<", "2.0"))
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("integer", "number", "string")...)
}

func (s *NumericValueTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("numeric", true, func() rql.ASTNode {
			return NumericValue(Numeric("", s.N("0")))
		})
	}

	ast := s.A("number", s.A("<", "1"))
	s.EVFTC(ast, float64(1))
	s.EVTTC(ast, float64(0))
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("integer", "number", "string")...)
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.EVTTC(notAST, float64(1))
	s.EVFTC(notAST, float64(0))
	s.EVSTTC(notAST, s.VS("object", "array", "integer", "number", "string")...)
}

func TestNumericValue(t *testing.T) {
	s := new(NumericValueTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return NumericValue(Numeric("", s.N("0")))
	}
	suite.Run(t, s)
}
