package primary

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
)

// This test suite's a common base class for the time attribute
// primary unit tests.
type TimeAttrTestSuite struct {
	asttest.Suite
	name       string
	constructP func(rql.TimePredicate) rql.Primary
	setAttr    func(*rql.Entry, time.Time)
}

func (s *TimeAttrTestSuite) TestMarshal() {
	s.MTC(s.constructP(predicate.Time(predicate.LT, s.TM(1000))), s.A(s.name, s.A("<", s.TM(1000))))
}

func (s *TimeAttrTestSuite) TestUnmarshal() {
	p := s.constructP(predicate.Time("", s.TM(0)))
	s.UMETC(p, "foo", fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), true)
	s.UMETC(p, s.A(s.name, "foo", "bar"), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), false)
	s.UMETC(p, s.A(s.name), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate.*missing.*NPE TimePredicate`, s.name, s.name), false)
	s.UMETC(p, s.A(s.name, s.A("<", true)), fmt.Sprintf(`%v.*NPE TimePredicate.*valid.*time.*type`, s.name), false)
	s.UMTC(p, s.A(s.name, s.A("<", int64(1000))), s.constructP(predicate.Time(predicate.LT, s.TM(1000))))
}

func (s *TimeAttrTestSuite) TestEvalEntry() {
	p := s.constructP(predicate.Time(predicate.LT, s.TM(1000)))
	e := rql.Entry{}
	s.setAttr(&e, s.TM(2000))
	s.EEFTC(p, e)
	s.setAttr(&e, s.TM(500))
	s.EETTC(p, e)
}

func (s *TimeAttrTestSuite) TestExpression_Atom() {
	expr := expression.New(s.name, false, func() rql.ASTNode {
		return s.constructP(predicate.Time("", time.Time{}))
	})

	s.MUM(expr, []interface{}{s.name, []interface{}{"<", float64(1000)}})
	e := rql.Entry{}
	s.setAttr(&e, s.TM(2000))
	s.EEFTC(expr, e)
	s.setAttr(&e, s.TM(500))
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}
