package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type KindTestSuite struct {
	asttest.Suite
}

func (s *KindTestSuite) TestMarshal() {
	s.MTC(Kind(predicate.StringGlob("foo")), s.A("kind", s.A("glob", "foo")))
}

func (s *KindTestSuite) TestUnmarshal() {
	p := Kind(predicate.StringGlob(""))
	s.UMETC(p, "foo", "formatted.*'kind'.*<string_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("glob", "foo")), "formatted.*'kind'.*<string_predicate>", true)
	s.UMETC(p, s.A("kind", "foo", "bar"), "formatted.*'kind'.*<string_predicate>", false)
	s.UMETC(p, s.A("kind"), "missing.*string.*predicate", false)
	s.UMETC(p, s.A("kind", s.A("glob", "[")), "glob", false)
	s.UMTC(p, s.A("kind", s.A("glob", "foo")), Kind(predicate.StringGlob("foo")))
}

func (s *KindTestSuite) TestEntrySchemaInDomain() {
	p := Kind(predicate.StringGlob("foo"))
	schema := &rql.EntrySchema{}
	s.ESIDFTC(p, schema)
	schema.SetPath("bar")
	s.ESIDTTC(p, schema)
}

func (s *KindTestSuite) TestEvalEntrySchema() {
	p := Kind(predicate.StringGlob("foo"))
	schema := &rql.EntrySchema{}
	schema.SetPath("bar")
	s.EESFTC(p, schema)
	schema.SetPath("foo")
	s.EESTTC(p, schema)
}

func (s *KindTestSuite) TestEntryInDomain() {
	p := Kind(predicate.StringGlob("foo"))
	e := rql.Entry{}
	s.EIDFTC(p, e)
	e.Schema = &rql.EntrySchema{}
	s.EIDTTC(p, e)
}

func (s *KindTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("kind", func() rql.ASTNode {
		return Kind(predicate.String())
	})

	s.MUM(expr, []interface{}{"kind", []interface{}{"glob", "foo"}})
	// Note from the semantics that EvalEntry(e) == EntryInDomain(e)
	e := rql.Entry{}
	s.EEFTC(expr, e)
	e.Schema = &rql.EntrySchema{}
	e.Schema.SetPath("bar")
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	schema.SetPath("")
	s.EESFTC(expr, schema)
	schema.SetPath("bar")
	s.EESFTC(expr, schema)
	schema.SetPath("foo")
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"kind", []interface{}{"glob", "foo"}}})

	// Remember that EvalEntry(e) == EntryInDomain(e)
	e.Schema = nil
	s.EEFTC(expr, e)
	e.Schema = &rql.EntrySchema{}
	e.Schema.SetPath("bar")
	s.EETTC(expr, e)

	schema.SetPath("")
	s.EESFTC(expr, schema)
	schema.SetPath("bar")
	s.EESTTC(expr, schema)
	schema.SetPath("foo")
	s.EESFTC(expr, schema)
}

func TestKind(t *testing.T) {
	suite.Run(t, new(KindTestSuite))
}
