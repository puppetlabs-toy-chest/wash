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

type CrtimeTestSuite struct {
	asttest.Suite
}

func (s *CrtimeTestSuite) TestMarshal() {
	s.MTC(Crtime(predicate.Time(predicate.LT, s.TM(1000))), s.A("crtime", s.A("<", s.TM(1000))))
}

func (s *CrtimeTestSuite) TestUnmarshal() {
	p := Crtime(predicate.Time("", s.TM(0)))
	s.UMETC(p, "foo", "formatted.*'crtime'.*<time_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), "formatted.*'crtime'.*<time_predicate>", true)
	s.UMETC(p, s.A("crtime", "foo", "bar"), "formatted.*'crtime'.*<time_predicate>", false)
	s.UMETC(p, s.A("crtime"), "missing.*time.*predicate", false)
	s.UMETC(p, s.A("crtime", s.A("<", true)), "valid.*time.*type", false)
	s.UMTC(p, s.A("crtime", s.A("<", int64(1000))), Crtime(predicate.Time(predicate.LT, s.TM(1000))))
}

func (s *CrtimeTestSuite) TestEntryInDomain() {
	p := Crtime(predicate.Time(predicate.LT, s.TM(1000)))
	s.EIDTTC(p, rql.Entry{})
}

func (s *CrtimeTestSuite) TestEvalEntry() {
	p := Crtime(predicate.Time(predicate.LT, s.TM(1000)))
	e := rql.Entry{}
	e.Attributes.SetCrtime(s.TM(2000))
	s.EEFTC(p, e)
	e.Attributes.SetCrtime(s.TM(500))
	s.EETTC(p, e)
}

func (s *CrtimeTestSuite) TestEntrySchemaInDomain() {
	p := Crtime(predicate.Time(predicate.LT, s.TM(1000)))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *CrtimeTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("crtime", func() rql.ASTNode {
		return Crtime(predicate.Time("", time.Time{}))
	})

	s.MUM(expr, []interface{}{"crtime", []interface{}{"<", float64(1000)}})
	e := rql.Entry{}
	e.Attributes.SetCrtime(s.TM(2000))
	s.EEFTC(expr, e)
	e.Attributes.SetCrtime(s.TM(500))
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"crtime", []interface{}{"<", float64(1000)}}})
	e.Attributes.SetCrtime(s.TM(2000))
	s.EETTC(expr, e)
	e.Attributes.SetCrtime(s.TM(500))
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestCrtime(t *testing.T) {
	suite.Run(t, new(CrtimeTestSuite))
}
