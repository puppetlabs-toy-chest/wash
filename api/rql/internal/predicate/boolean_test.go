package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type BooleanTestSuite struct {
	asttest.Suite
}

func (s *BooleanTestSuite) TestMarshal() {
	s.MTC(Boolean(true), s.A("boolean", true))
	s.MTC(Boolean(false), s.A("boolean", false))
}

func (s *BooleanTestSuite) TestUnmarshal() {
	b := Boolean(true)
	s.UMETC(b, "foo", "formatted.*boolean.*value", true)
	s.UMETC(b, s.A("boolean", "foo", "bar"), "formatted.*boolean.*value", false)
	s.UMETC(b, s.A("boolean"), "formatted.*boolean.*value.*missing.*value", false)
	s.UMETC(b, s.A("boolean", "foo"), "foo.*valid.*Boolean.*true.*false", true)
	s.UMTC(b, s.A("boolean", true), Boolean(true))
	s.UMTC(b, s.A("boolean", false), Boolean(false))
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

func (s *BooleanTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("boolean", true, func() rql.ASTNode {
		return Boolean(false)
	})

	s.MUM(expr, s.A("boolean", true))
	s.EVFTC(expr, false)
	s.EVTTC(expr, true)
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", s.A("boolean", true)})
	s.EVTTC(expr, false)
	s.EVFTC(expr, true)
}

func TestBoolean(t *testing.T) {
	suite.Run(t, new(BooleanTestSuite))
}
