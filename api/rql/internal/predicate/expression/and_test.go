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

type AndTestSuite struct {
	asttest.Suite
}

func (s *AndTestSuite) TestMarshal() {
	p := And(predicate.Boolean(true), predicate.Boolean(false))
	s.MTC(p, s.A("AND", predicate.Boolean(true).Marshal(), predicate.Boolean(false).Marshal()))
}

func (s *AndTestSuite) TestUnmarshal() {
	p := And(predicate.Boolean(false), predicate.Boolean(false))
	s.UMETC(p, "foo", `formatted.*"AND".*<pe>.*<pe>`, true)
	s.UMETC(p, s.A("AND", "foo", "bar", "baz"), `"AND".*<pe>.*<pe>`, false)
	s.UMETC(p, s.A("AND"), "AND.*LHS.*RHS.*expression", false)
	s.UMETC(p, s.A("AND", true), "AND.*LHS.*RHS.*expression", false)
	s.UMETC(p, s.A("AND", "foo", true), "AND.*LHS.*Boolean", false)
	s.UMETC(p, s.A("AND", true, "foo"), "AND.*RHS.*Boolean", false)
	s.UMTC(p, s.A("AND", true, true), And(predicate.Boolean(true), predicate.Boolean(true)))
}

func (s *AndTestSuite) TestEvalEntry() {
	s.EEFTC(And(primary.Boolean(false), primary.Boolean(false)), rql.Entry{})
	s.EEFTC(And(primary.Boolean(false), primary.Boolean(true)), rql.Entry{})
	s.EEFTC(And(primary.Boolean(true), primary.Boolean(false)), rql.Entry{})
	s.EETTC(And(primary.Boolean(true), primary.Boolean(true)), rql.Entry{})
}

func (s *AndTestSuite) TestEvalEntrySchema() {
	s.EESFTC(And(primary.Boolean(false), primary.Boolean(false)), &rql.EntrySchema{})
	s.EESFTC(And(primary.Boolean(false), primary.Boolean(true)), &rql.EntrySchema{})
	s.EESFTC(And(primary.Boolean(true), primary.Boolean(false)), &rql.EntrySchema{})
	s.EESTTC(And(primary.Boolean(true), primary.Boolean(true)), &rql.EntrySchema{})
}

func (s *AndTestSuite) TestEvalValue() {
	// Note that we can't use predicate.Boolean(val) here because those return true if v == val
	p := And(predicate.NumericValue(predicate.LT, s.N("10")), predicate.NumericValue(predicate.GT, s.N("10")))
	// p1 == false, p2 == false
	s.EVFTC(p, float64(10))
	// false, true
	s.EVFTC(p, float64(11))
	// true, false
	s.EVFTC(p, float64(9))
	// true, true
	p.(*and).p2 = p.(*and).p1
	s.EVTTC(p, float64(9))
}

func (s *AndTestSuite) TestEvalString() {
	p := And(predicate.StringValueEqual("one"), predicate.StringValueEqual("two"))
	// p1 == false, p2 == false
	s.ESFTC(p, "foo")
	// false, true
	s.ESFTC(p, "two")
	// true, false
	s.ESFTC(p, "one")
	// true, true
	p.(*and).p2 = p.(*and).p1
	s.ESTTC(p, "one")
}

func (s *AndTestSuite) TestEvalNumeric() {
	p := And(predicate.NumericValue(predicate.LT, s.N("10")), predicate.NumericValue(predicate.GT, s.N("10")))
	// p1 == false, p2 == false
	s.ENFTC(p, s.N("10"))
	// false, true
	s.ENFTC(p, s.N("11"))
	// true, false
	s.ENFTC(p, s.N("9"))
	// true, true
	p.(*and).p2 = p.(*and).p1
	s.ENTTC(p, s.N("9"))
}

func (s *AndTestSuite) TestEvalTime() {
	p := And(predicate.TimeValue(predicate.LT, s.TM(10)), predicate.TimeValue(predicate.GT, s.TM(10)))
	// p1 == false, p2 == false
	s.ETFTC(p, s.TM(10))
	// false, true
	s.ETFTC(p, s.TM(11))
	// true, false
	s.ETFTC(p, s.TM(9))
	// true, true
	p.(*and).p2 = p.(*and).p1
	s.ETTTC(p, s.TM(9))
}

func (s *AndTestSuite) TestEvalAction() {
	p := And(predicate.Action(plugin.ExecAction()), predicate.Action(plugin.ListAction()))
	// p1 == false, p2 == false
	s.EAFTC(p, plugin.DeleteAction())
	// false, true
	s.EAFTC(p, plugin.ListAction())
	// true, false
	s.EAFTC(p, plugin.ExecAction())
	// true, true
	p.(*and).p2 = p.(*and).p1
	s.EATTC(p, plugin.ExecAction())
}

func TestAnd(t *testing.T) {
	suite.Run(t, new(AndTestSuite))
}
