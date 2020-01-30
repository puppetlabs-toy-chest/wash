package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type NameTestSuite struct {
	asttest.Suite
}

func (s *NameTestSuite) TestMarshal() {
	s.MTC(Name(predicate.StringGlob("foo")), s.A("name", s.A("glob", "foo")))
}

func (s *NameTestSuite) TestUnmarshal() {
	n := Name(predicate.StringGlob(""))
	s.UMETC(n, "foo", "formatted.*'name'.*<string_predicate>", true)
	s.UMETC(n, s.A("foo", s.A("glob", "foo")), "formatted.*'name'.*<string_predicate>", true)
	s.UMETC(n, s.A("name", "foo", "bar"), "formatted.*'name'.*<string_predicate>", false)
	s.UMETC(n, s.A("name"), "missing.*string.*predicate", false)
	s.UMETC(n, s.A("name", s.A("glob", "[")), "glob", false)
	s.UMTC(n, s.A("name", s.A("glob", "foo")), Name(predicate.StringGlob("foo")))
}

func (s *NameTestSuite) TestEntryInDomain() {
	p := Name(predicate.StringGlob("foo"))
	s.EIDTTC(p, rql.Entry{})
}

func (s *NameTestSuite) TestEvalEntry() {
	n := Name(predicate.StringGlob("foo"))
	e := rql.Entry{}
	e.Name = "bar"
	s.EEFTC(n, e)
	e.Name = "foo"
	s.EETTC(n, e)
}

func (s *NameTestSuite) TestEntrySchemaInDomain() {
	p := Name(predicate.StringGlob("foo"))
	s.ESIDTTC(p, &rql.EntrySchema{})
}

func (s *NameTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("name", func() rql.ASTNode {
		return Name(predicate.String())
	})

	s.MUM(expr, []interface{}{"name", []interface{}{"glob", "foo"}})
	e := rql.Entry{}
	e.Name = "bar"
	s.EEFTC(expr, e)
	e.Name = "foo"
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"name", []interface{}{"glob", "foo"}}})
	e.Name = "bar"
	s.EETTC(expr, e)
	e.Name = "foo"
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestName(t *testing.T) {
	suite.Run(t, new(NameTestSuite))
}
