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

func (s *NumericTestSuite) TestNumeric_Marshal() {
	s.MTC(Numeric(LT, s.N("2.3")), s.A("<", "2.3"))
}

func (s *NumericTestSuite) TestNumeric_Unmarshal() {
	n := Numeric("", s.N("0"))
	s.UMETC(n, "foo", "formatted.*<comparison_op>.*<number>", true)
	s.UMETC(n, s.A(), "formatted.*<comparison_op>.*<number>", true)
	s.UMETC(n, s.A("<", "foo", "bar"), "formatted.*<comparison_op>.*<number>", false)
	s.UMETC(n, s.A("<"), "formatted.*<comparison_op>.*<number>.*missing.*number", false)
	s.UMETC(n, s.A("<", true), "valid.*number", false)
	s.UMETC(n, s.A("<", "true"), "parse.*true.*number.*exponent", false)
	s.UMTC(n, s.A("<", 2.3), Numeric(LT, s.N("2.3")))
	s.UMTC(n, s.A("<", "2.3"), Numeric(LT, s.N("2.3")))
	// Test unmarshaling a very large value
	largeV := "10000000000000000000000000000000000000000000000000000000000000000000"
	s.UMTC(n, s.A("<", largeV), Numeric(LT, s.N(largeV)))
}

func (s *NumericTestSuite) TestNumeric_UnmarshalUnsigned() {
	n := UnsignedNumeric("", s.N("0"))
	s.UMETC(n, s.A("<", "-10"), "unsigned.*number", false)
}

func (s *NumericTestSuite) TestNumeric_EvalNumeric() {
	// Test LT
	n := Numeric(LT, s.N("1"))
	s.ENFTC(n, s.N("2"), s.N("1"))
	s.ENTTC(n, s.N("0"))

	// Test LTE
	n = Numeric(LTE, s.N("1"))
	s.ENFTC(n, s.N("2"))
	s.ENTTC(n, s.N("0"), s.N("1"))

	// Test GT
	n = Numeric(GT, s.N("1"))
	s.ENFTC(n, s.N("0"), s.N("1"))
	s.ENTTC(n, s.N("2"))

	// Test GTE
	n = Numeric(GTE, s.N("1"))
	s.ENFTC(n, s.N("0"))
	s.ENTTC(n, s.N("2"), s.N("1"))

	// Test EQL
	n = Numeric(EQL, s.N("1"))
	s.ENFTC(n, s.N("0"), s.N("2"))
	s.ENTTC(n, s.N("1"))

	// TEST NEQL
	n = Numeric(NEQL, s.N("1"))
	s.ENFTC(n, s.N("1"))
	s.ENTTC(n, s.N("0"), s.N("2"))
}

func (s *NumericTestSuite) TestNumeric_Expression_AtomAndNot() {
	expr := expression.New("numeric", true, func() rql.ASTNode {
		return Numeric("", s.N("0"))
	})

	s.MUM(expr, []interface{}{"<", "1"})
	s.ENFTC(expr, s.N("1"))
	s.ENTTC(expr, s.N("0"))
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"<", "1"}})
	s.ENTTC(expr, s.N("1"))
	s.ENFTC(expr, s.N("0"))
}

func (s *NumericTestSuite) TestNumericValue_Marshal() {
	s.MTC(NumericValue(LT, s.N("2.3")), s.A("number", s.A("<", "2.3")))
}

func (s *NumericTestSuite) TestNumericValue_Unmarshal() {
	n := NumericValue("", s.N("0"))
	s.UMETC(n, "foo", "formatted.*number.*<numeric_predicate>", true)
	s.UMETC(n, s.A("number", "foo", "bar"), "formatted.*number.*<numeric_predicate>", false)
	s.UMETC(n, s.A("number"), "formatted.*number.*<numeric_predicate>.*missing.*numeric.*predicate", false)
	s.UMETC(n, s.A("number", s.A()), "unmarshalling.*numeric.*predicate.*formatted.*<comparison_op>.*<number>", false)
	s.UMTC(n, s.A("number", s.A("<", "2.3")), NumericValue(LT, s.N("2.3")))
}

func (s *NumericTestSuite) TestNumericValue_EvalValue() {
	n := NumericValue(LT, s.N("2.0"))
	s.EVFTC(n, float64(3))
	s.EVTTC(n, float64(1))
	// TestEvalNumeric contained the operator-specific test-cases
}

func (s *NumericTestSuite) TestNumericValue_Expression_AtomAndNot() {
	expr := expression.New("numeric", true, func() rql.ASTNode {
		return NumericValue("", s.N("0"))
	})

	s.MUM(expr, []interface{}{"number", []interface{}{"<", "1"}})
	s.EVFTC(expr, float64(1), "1")
	s.EVTTC(expr, float64(0))
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"number", []interface{}{"<", "1"}}})
	s.EVTTC(expr, float64(1), "1")
	s.EVFTC(expr, float64(0))
}

func TestNumeric(t *testing.T) {
	suite.Run(t, new(NumericTestSuite))
}
