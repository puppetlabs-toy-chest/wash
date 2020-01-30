package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type SizeTestSuite struct {
	asttest.Suite
}

func (s *SizeTestSuite) TestMarshal() {
	s.MTC(Size(UnsignedNumeric(LT, s.N("10"))), s.A("size", s.A("<", "10")))
}

func (s *SizeTestSuite) TestUnmarshal() {
	p := Size(UnsignedNumeric("", s.N("0")))
	s.UMETC(p, "foo", "formatted.*'size'.*<numeric_predicate>", true)
	s.UMETC(p, s.A("foo"), "formatted.*'size'.*<numeric_predicate>", true)
	s.UMETC(p, s.A("size", "foo", "bar"), "formatted.*'size'.*<numeric_predicate>", false)
	s.UMETC(p, s.A("size"), "missing.*predicate.*expression", false)
	s.UMETC(p, s.A("size", s.A("<", true)), "valid.*number", false)
	s.UMETC(p, s.A("size", s.A("<", "-10")), "unsigned.*number", false)
	s.UMTC(p, s.A("size", s.A("<", "10")), Size(UnsignedNumeric(LT, s.N("10"))))
}

func (s *SizeTestSuite) TestValueInDomain() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	s.VIDFTC(p, "foo", true)
	s.VIDTTC(p, map[string]interface{}{}, []interface{}{})
}

func (s *SizeTestSuite) EvalValue() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	s.EVFTC(p, map[string]interface{}{}, []interface{}{})
	s.EVTTC(p, map[string]interface{}{"foo": "bar"}, []interface{}{"foo"})
}

func (s *SizeTestSuite) TestEntryInDomain() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	s.EIDTTC(p, rql.Entry{})
}

func (s *SizeTestSuite) TestEvalEntry() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	e := rql.Entry{}
	e.Attributes.SetSize(uint64(0))
	s.EEFTC(p, e)
	e.Attributes.SetSize(uint64(1))
	s.EETTC(p, e)
}

func (s *SizeTestSuite) TestEntrySchemaInDomain() {
	p := Size(UnsignedNumeric(GT, s.N("0")))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *SizeTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("size", func() rql.ASTNode {
		return Size(UnsignedNumeric("", s.N("0")))
	})

	s.MUM(expr, []interface{}{"size", []interface{}{">", "0"}})
	s.EVFTC(expr, map[string]interface{}{}, []interface{}{}, "foo")
	s.EVTTC(expr, map[string]interface{}{"foo": "bar"}, []interface{}{"foo"})

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

	// Test Not
	s.MUM(expr, []interface{}{"NOT", []interface{}{"size", []interface{}{">", "0"}}})
	s.EVTTC(expr, map[string]interface{}{}, []interface{}{})
	s.EVFTC(expr, map[string]interface{}{"foo": "bar"}, []interface{}{"foo"}, "foo")

	e.Attributes.SetSize(uint64(0))
	s.EETTC(expr, e)
	e.Attributes.SetSize(uint64(1))
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestSize(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}
