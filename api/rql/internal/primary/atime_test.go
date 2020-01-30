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

type AtimeTestSuite struct {
	asttest.Suite
}

func (s *AtimeTestSuite) TestMarshal() {
	s.MTC(Atime(predicate.Time(predicate.LT, s.TM(1000))), s.A("atime", s.A("<", s.TM(1000))))
}

func (s *AtimeTestSuite) TestUnmarshal() {
	p := Atime(predicate.Time("", s.TM(0)))
	s.UMETC(p, "foo", "formatted.*'atime'.*<time_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("<", int64(1000))), "formatted.*'atime'.*<time_predicate>", true)
	s.UMETC(p, s.A("atime", "foo", "bar"), "formatted.*'atime'.*<time_predicate>", false)
	s.UMETC(p, s.A("atime"), "missing.*time.*predicate", false)
	s.UMETC(p, s.A("atime", s.A("<", true)), "valid.*time.*type", false)
	s.UMTC(p, s.A("atime", s.A("<", int64(1000))), Atime(predicate.Time(predicate.LT, s.TM(1000))))
}

func (s *AtimeTestSuite) TestEntryInDomain() {
	p := Atime(predicate.Time(predicate.LT, s.TM(1000)))
	s.EIDTTC(p, rql.Entry{})
}

func (s *AtimeTestSuite) TestEvalEntry() {
	p := Atime(predicate.Time(predicate.LT, s.TM(1000)))
	e := rql.Entry{}
	e.Attributes.SetAtime(s.TM(2000))
	s.EEFTC(p, e)
	e.Attributes.SetAtime(s.TM(500))
	s.EETTC(p, e)
}

func (s *AtimeTestSuite) TestEntrySchemaInDomain() {
	p := Atime(predicate.Time(predicate.LT, s.TM(1000)))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *AtimeTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("atime", func() rql.ASTNode {
		return Atime(predicate.Time("", time.Time{}))
	})

	s.MUM(expr, []interface{}{"atime", []interface{}{"<", float64(1000)}})
	e := rql.Entry{}
	e.Attributes.SetAtime(s.TM(2000))
	s.EEFTC(expr, e)
	e.Attributes.SetAtime(s.TM(500))
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"atime", []interface{}{"<", float64(1000)}}})
	e.Attributes.SetAtime(s.TM(2000))
	s.EETTC(expr, e)
	e.Attributes.SetAtime(s.TM(500))
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestAtime(t *testing.T) {
	suite.Run(t, new(AtimeTestSuite))
}
