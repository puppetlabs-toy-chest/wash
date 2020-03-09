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
	name                string
	timeNodeConstructor func(p rql.TimePredicate) rql.Primary
	setAttr             func(*rql.Entry, time.Time)
}

func newTimeAttrTestSuite(
	name string,
	timeNodeConstructor func(p rql.TimePredicate) rql.Primary,
	setAttr func(*rql.Entry, time.Time),
) *TimeAttrTestSuite {
	s := new(TimeAttrTestSuite)
	s.name = name
	s.timeNodeConstructor = timeNodeConstructor
	s.setAttr = setAttr
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return s.timeNodeConstructor(predicate.Time("", s.TM(0)))
	}
	return s
}

func (s *TimeAttrTestSuite) TestMarshal() {
	s.MTC(s.timeNodeConstructor(predicate.Time("<", s.TM(1000))), s.A(s.name, s.A("<", s.TM(1000))))
}

func (s *TimeAttrTestSuite) TestUnmarshal() {
	s.UMETC("foo", fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), true)
	s.UMETC(s.A("foo", s.A("<", int64(1000))), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), true)
	s.UMETC(s.A(s.name, "foo", "bar"), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate`, s.name, s.name), false)
	s.UMETC(s.A(s.name), fmt.Sprintf(`%v.*formatted.*"%v".*NPE TimePredicate.*missing.*NPE TimePredicate`, s.name, s.name), false)
	s.UMETC(s.A(s.name, s.A("<", true)), fmt.Sprintf(`%v.*NPE TimePredicate.*valid.*time.*type`, s.name), false)
}

func (s *TimeAttrTestSuite) TestEvalEntry() {
	ast := s.A(s.name, s.A("<", s.TM(1000)))
	e := rql.Entry{}
	s.setAttr(&e, s.TM(2000))
	s.EEFTC(ast, e)
	s.setAttr(&e, s.TM(500))
	s.EETTC(ast, e)
}

func (s *TimeAttrTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New(s.name, false, func() rql.ASTNode {
			return s.DefaultNodeConstructor()
		})
	}

	ast := s.A(s.name, s.A("<", 1000))
	e := rql.Entry{}
	s.setAttr(&e, s.TM(2000))
	s.EEFTC(ast, e)
	s.setAttr(&e, s.TM(500))
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
	s.EESTTC(ast, schema)

	s.AssertNotImplemented(
		ast,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}
