package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/primary"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type OrTestSuite struct {
	asttest.Suite
}

func (s *OrTestSuite) TestMarshal() {
	p := Or(predicate.Boolean(true), predicate.Boolean(false))
	s.MTC(p, s.A("OR", predicate.Boolean(true).Marshal(), predicate.Boolean(false).Marshal()))
}

func (s *OrTestSuite) TestUnmarshal() {
	p := Or(predicate.Boolean(false), predicate.Boolean(false))
	s.UMETC(p, "foo", "formatted.*'OR'.*<pe>.*<pe>", true)
	s.UMETC(p, s.A("OR", "foo", "bar", "baz"), "'OR'.*<pe>.*<pe>", false)
	s.UMETC(p, s.A("OR"), "OR.*LHS.*RHS.*expression", false)
	s.UMETC(p, s.A("OR", true), "OR.*LHS.*RHS.*expression", false)
	s.UMETC(p, s.A("OR", "foo", true), "OR.*LHS.*<boolean_value>", false)
	s.UMETC(p, s.A("OR", true, "foo"), "OR.*RHS.*<boolean_value>", false)
	s.UMTC(p, s.A("OR", true, true), Or(predicate.Boolean(true), predicate.Boolean(true)))
}

func (s *OrTestSuite) TestEvalEntry() {
	s.EEFTC(Or(primary.Boolean(false), primary.Boolean(false)), rql.Entry{})
	s.EETTC(Or(primary.Boolean(false), primary.Boolean(true)), rql.Entry{})
	s.EETTC(Or(primary.Boolean(true), primary.Boolean(false)), rql.Entry{})
	s.EETTC(Or(primary.Boolean(true), primary.Boolean(true)), rql.Entry{})
}

func (s *OrTestSuite) TestEvalEntrySchema() {
	s.EESFTC(Or(primary.Boolean(false), primary.Boolean(false)), &rql.EntrySchema{})
	s.EESTTC(Or(primary.Boolean(false), primary.Boolean(true)), &rql.EntrySchema{})
	s.EESTTC(Or(primary.Boolean(true), primary.Boolean(false)), &rql.EntrySchema{})
	s.EESTTC(Or(primary.Boolean(true), primary.Boolean(true)), &rql.EntrySchema{})
}

func (s *OrTestSuite) TestEvalValue() {
	// Note that we can't use predicate.Boolean(val) here because those return true if v == val
	p := Or(predicate.NumericValue(predicate.LT, s.N("10")), predicate.NumericValue(predicate.GT, s.N("10")))
	// p1 == false, p2 == false
	s.EVFTC(p, float64(10))
	// false, true
	s.EVTTC(p, float64(11))
	// true, false
	s.EVTTC(p, float64(9))
	// true, true
	p.(*or).p2 = p.(*or).p1
	s.EVTTC(p, float64(9))
}

func (s *OrTestSuite) TestEvalString() {
	p := Or(predicate.StringValueEqual("one"), predicate.StringValueEqual("two"))
	// p1 == false, p2 == false
	s.ESFTC(p, "foo")
	// false, true
	s.ESTTC(p, "two")
	// true, false
	s.ESTTC(p, "one")
	// true, true
	p.(*or).p2 = p.(*or).p1
	s.ESTTC(p, "one")
}

func (s *OrTestSuite) TestEvalNumeric() {
	p := Or(predicate.NumericValue(predicate.LT, s.N("10")), predicate.NumericValue(predicate.GT, s.N("10")))
	// p1 == false, p2 == false
	s.ENFTC(p, s.N("10"))
	// false, true
	s.ENTTC(p, s.N("11"))
	// true, false
	s.ENTTC(p, s.N("9"))
	// true, true
	p.(*or).p2 = p.(*or).p1
	s.ENTTC(p, s.N("9"))
}

func (s *OrTestSuite) TestEvalTime() {
	p := Or(predicate.TimeValue(predicate.LT, s.TM(10)), predicate.TimeValue(predicate.GT, s.TM(10)))
	// p1 == false, p2 == false
	s.ETFTC(p, s.TM(10))
	// false, true
	s.ETTTC(p, s.TM(11))
	// true, false
	s.ETTTC(p, s.TM(9))
	// true, true
	p.(*or).p2 = p.(*or).p1
	s.ETTTC(p, s.TM(9))
}

func (s *OrTestSuite) TestEvalAction() {
	p := Or(predicate.Action(plugin.ExecAction()), predicate.Action(plugin.ListAction()))
	// p1 == false, p2 == false
	s.EAFTC(p, plugin.DeleteAction())
	// false, true
	s.EATTC(p, plugin.ListAction())
	// true, false
	s.EATTC(p, plugin.ExecAction())
	// true, true
	p.(*or).p2 = p.(*or).p1
	s.EATTC(p, plugin.ExecAction())
}

func TestOr(t *testing.T) {
	suite.Run(t, new(OrTestSuite))
}
