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

func (s *BooleanTestSuite) TestUnmarshal() {
	b := Boolean(true)
	s.UMETC(b, "foo", "foo.*valid.*Boolean.*true.*false", true)
	s.UMTC(b, true, Boolean(true))
	s.UMTC(b, false, Boolean(false))
}

func (s *BooleanTestSuite) TestEvalValue() {
	// Test true
	b := Boolean(true)
	s.EVFTC(b, false, "foo")
	s.EVTTC(b, true)

	// Test false
	b = Boolean(false)
	s.EVFTC(b, true, "foo")
	s.EVTTC(b, false)
}

func (s BooleanTestSuite) TestEvalValueSchema() {
	b := Boolean(true)
	s.EVSFTC(b, s.VS("object", "array")...)
	s.EVSTTC(b, s.VS("boolean")...)
}

func (s *BooleanTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("boolean", true, func() rql.ASTNode {
		return Boolean(false)
	})

	s.MUM(expr, true)
	s.EVFTC(expr, false)
	s.EVTTC(expr, true)
	s.EVSFTC(expr, s.VS("object", "array")...)
	s.EVSTTC(expr, s.VS("boolean")...)
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", true})
	s.EVTTC(expr, false)
	s.EVFTC(expr, true)
	s.EVSTTC(expr, s.VS("object", "array", "boolean")...)
}

func TestBoolean(t *testing.T) {
	suite.Run(t, new(BooleanTestSuite))
}
