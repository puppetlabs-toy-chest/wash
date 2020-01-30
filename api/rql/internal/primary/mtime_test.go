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

type MtimeTestSuite struct {
	asttest.Suite
}

func (s *MtimeTestSuite) TestMarshal() {
	s.MTC(Mtime(predicate.Time(predicate.LT, s.TM(1000))), s.A("mtime", s.A("<", s.TM(1000))))
}

func (s *MtimeTestSuite) TestUnmarshal() {
	p := Mtime(predicate.Time("", s.TM(0)))
	s.UMETC(p, "foo", "formatted.*'mtime'.*<time_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), "formatted.*'mtime'.*<time_predicate>", true)
	s.UMETC(p, s.A("mtime", "foo", "bar"), "formatted.*'mtime'.*<time_predicate>", false)
	s.UMETC(p, s.A("mtime"), "missing.*time.*predicate", false)
	s.UMETC(p, s.A("mtime", s.A("<", true)), "valid.*time.*type", false)
	s.UMTC(p, s.A("mtime", s.A("<", int64(1000))), Mtime(predicate.Time(predicate.LT, s.TM(1000))))
}

func (s *MtimeTestSuite) TestEntryInDomain() {
	p := Mtime(predicate.Time(predicate.LT, s.TM(1000)))
	s.EIDTTC(p, rql.Entry{})
}

func (s *MtimeTestSuite) TestEvalEntry() {
	p := Mtime(predicate.Time(predicate.LT, s.TM(1000)))
	e := rql.Entry{}
	e.Attributes.SetMtime(s.TM(2000))
	s.EEFTC(p, e)
	e.Attributes.SetMtime(s.TM(500))
	s.EETTC(p, e)
}

func (s *MtimeTestSuite) TestEntrySchemaInDomain() {
	p := Mtime(predicate.Time(predicate.LT, s.TM(1000)))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *MtimeTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("mtime", func() rql.ASTNode {
		return Mtime(predicate.Time("", time.Time{}))
	})

	s.MUM(expr, []interface{}{"mtime", []interface{}{"<", float64(1000)}})
	e := rql.Entry{}
	e.Attributes.SetMtime(s.TM(2000))
	s.EEFTC(expr, e)
	e.Attributes.SetMtime(s.TM(500))
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"mtime", []interface{}{"<", float64(1000)}}})
	e.Attributes.SetMtime(s.TM(2000))
	s.EETTC(expr, e)
	e.Attributes.SetMtime(s.TM(500))
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestMtime(t *testing.T) {
	suite.Run(t, new(MtimeTestSuite))
}
