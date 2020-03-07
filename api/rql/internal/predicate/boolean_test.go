package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type BooleanTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *BooleanTestSuite) TestMarshal() {
	s.MTC(Boolean(true), true)
	s.MTC(Boolean(false), false)
}

func (s *BooleanTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", "foo.*valid.*Boolean.*true.*false", true)
}

func (s *BooleanTestSuite) TestEvalValue() {
	// Test true
	s.EVFTC(true, false, "foo")
	s.EVTTC(true, true)

	// Test false
	s.EVFTC(false, true, "foo")
	s.EVTTC(false, false)
}

func (s *BooleanTestSuite) TestEvalValueSchema() {
	s.EVSFTC(true, s.VS("object", "array")...)
	s.EVSTTC(true, s.VS("boolean")...)
}

func (s *BooleanTestSuite) TestEvalEntry() {
	// Test true
	s.EETTC(true, rql.Entry{})
	// Test false
	s.EEFTC(false, rql.Entry{})
}

func (s *BooleanTestSuite) TestEvalEntrySchema() {
	// Test true
	s.EESTTC(true, &rql.EntrySchema{})
	// Test false
	s.EESFTC(false, &rql.EntrySchema{})
}

func (s *BooleanTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("boolean", true, func() rql.ASTNode {
			return Boolean(false)
		})
	}

	s.EVFTC(true, false)
	s.EVTTC(true, true)
	s.EVSFTC(true, s.VS("object", "array")...)
	s.EVSTTC(true, s.VS("boolean")...)
	s.EETTC(true, rql.Entry{})
	s.EESTTC(true, &rql.EntrySchema{})
	s.AssertNotImplemented(
		true,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	// Only for EvalValue and EvalValueSchema
	s.EVTTC(s.A("NOT", true), false)
	s.EVFTC(s.A("NOT", true), true)
	s.EVSTTC(s.A("NOT", true), s.VS("object", "array", "boolean")...)
}

func TestBoolean(t *testing.T) {
	s := new(BooleanTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Boolean(true)
	}
	suite.Run(t, s)
}
