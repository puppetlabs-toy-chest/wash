package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type CtimeTestSuite struct {
	asttest.Suite
}

func (s *CtimeTestSuite) TestMarshal() {
	s.MTC(Ctime(predicate.Time(predicate.LT, s.TM(1000))), s.A("ctime", s.A("<", s.TM(1000))))
}

func (s *CtimeTestSuite) TestUnmarshal() {
	p := Ctime(predicate.Time("", s.TM(0)))
	s.UMETC(p, "foo", "formatted.*'ctime'.*<time_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), "formatted.*'ctime'.*<time_predicate>", true)
	s.UMETC(p, s.A("ctime", "foo", "bar"), "formatted.*'ctime'.*<time_predicate>", false)
	s.UMETC(p, s.A("ctime"), "missing.*time.*predicate", false)
	s.UMETC(p, s.A("ctime", s.A("<", true)), "valid.*time.*type", false)
	s.UMTC(p, s.A("ctime", s.A("<", int64(1000))), Ctime(predicate.Time(predicate.LT, s.TM(1000))))
}

func (s *CtimeTestSuite) TestEntryInDomain() {
	p := Ctime(predicate.Time(predicate.LT, s.TM(1000)))
	s.EIDTTC(p, rql.Entry{})
}

func (s *CtimeTestSuite) TestEvalEntry() {
	p := Ctime(predicate.Time(predicate.LT, s.TM(1000)))
	e := rql.Entry{}
	e.Attributes.SetCtime(s.TM(2000))
	s.EEFTC(p, e)
	e.Attributes.SetCtime(s.TM(500))
	s.EETTC(p, e)
}

func (s *CtimeTestSuite) TestEntrySchemaInDomain() {
	p := Ctime(predicate.Time(predicate.LT, s.TM(1000)))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *CtimeTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("ctime", func() rql.ASTNode {
		return Ctime(predicate.Time("", time.Time{}))
	})

	s.MUM(expr, []interface{}{"ctime", []interface{}{"<", float64(1000)}})
	e := rql.Entry{}
	e.Attributes.SetCtime(s.TM(2000))
	s.EEFTC(expr, e)
	e.Attributes.SetCtime(s.TM(500))
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"ctime", []interface{}{"<", float64(1000)}}})
	e.Attributes.SetCtime(s.TM(2000))
	s.EETTC(expr, e)
	e.Attributes.SetCtime(s.TM(500))
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestCtime(t *testing.T) {
	suite.Run(t, new(CtimeTestSuite))
}
