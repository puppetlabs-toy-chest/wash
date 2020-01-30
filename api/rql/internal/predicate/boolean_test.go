package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

type BooleanTestSuite struct {
	asttest.Suite
}

func (s *BooleanTestSuite) TestMarshal() {
	s.MTC(Boolean(true), true)
	s.MTC(Boolean(false), false)
}

func (s *BooleanTestSuite) TestUnmarshal() {
	b := Boolean(true)
	s.UMETC(b, "foo", "formatted.*<boolean_value>", true)
	s.UMTC(b, true, Boolean(true))
	s.UMTC(b, false, Boolean(false))
}

func (s *BooleanTestSuite) TestValueInDomain() {
	// Test true
	b := Boolean(true)
	s.VIDFTC(b, "foo")
	s.VIDTTC(b, false)

	// Test false
	b = Boolean(false)
	s.VIDFTC(b, "foo")
	s.VIDTTC(b, true)
}

func (s *BooleanTestSuite) TestEvalValue() {
	// Test true
	b := Boolean(true)
	s.EVFTC(b, false)
	s.EVTTC(b, true)

	// Test false
	b = Boolean(false)
	s.EVFTC(b, true)
	s.EVTTC(b, false)
}

func (s *BooleanTestSuite) TestEntryInDomain() {
	// Test true
	b := Boolean(true)
	s.EIDTTC(b, rql.Entry{})

	// Test false
	b = Boolean(false)
	s.EIDTTC(b, rql.Entry{})
}

func (s *BooleanTestSuite) TestEvalEntry() {
	// Test true
	b := Boolean(true).(rql.EntryPredicate)
	s.EETTC(b, rql.Entry{})

	// Test false
	b = Boolean(false).(rql.EntryPredicate)
	s.EEFTC(b, rql.Entry{})
}

func (s *BooleanTestSuite) TestEntrySchemaInDomain() {
	// Test true
	b := Boolean(true)
	s.ESIDTTC(b, &rql.EntrySchema{})

	// Test false
	b = Boolean(false)
	s.ESIDTTC(b, &rql.EntrySchema{})
}

func (s *BooleanTestSuite) TestEvalEntrySchema() {
	// Test true
	b := Boolean(true).(rql.EntrySchemaPredicate)
	s.EESTTC(b, &apitypes.EntrySchema{})

	// Test false
	b = Boolean(false).(rql.EntrySchemaPredicate)
	s.EESFTC(b, &apitypes.EntrySchema{})
}

func (s *BooleanTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("boolean", func() rql.ASTNode {
		return Boolean(false)
	})

	s.MUM(expr, true)
	s.EVFTC(expr, false, "foo")
	s.EVTTC(expr, true)
	s.EETTC(expr, rql.Entry{})
	s.EESTTC(expr, &rql.EntrySchema{})
	s.AssertNotImplemented(
		expr,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", true})
	s.EVTTC(expr, false)
	s.EVFTC(expr, true, "foo")
	s.EEFTC(expr, rql.Entry{})
	s.EESFTC(expr, &rql.EntrySchema{})
}

func TestBoolean(t *testing.T) {
	suite.Run(t, new(BooleanTestSuite))
}
