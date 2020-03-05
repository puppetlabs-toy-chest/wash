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
	n := Kind(predicate.StringGlob(""))
	s.UMETC(n, "foo", `kind.*formatted.*"kind".*NPE StringPredicate`, true)
	s.UMETC(n, s.A("foo", s.A("glob", "foo")), `kind.*formatted.*"kind".*NPE StringPredicate`, true)
	s.UMETC(n, s.A("kind", "foo", "bar"), `kind.*formatted.*"kind".*NPE StringPredicate`, false)
	s.UMETC(n, s.A("kind"), `kind.*formatted.*"kind".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(n, s.A("kind", s.A("glob", "[")), "kind.*NPE StringPredicate.*glob", false)
	s.UMTC(n, s.A("kind", s.A("glob", "foo")), Kind(predicate.StringGlob("foo")))
}
func (s *KindTestSuite) TestEvalEntrySchema() {
	p := Kind(predicate.StringGlob("foo"))
	schema := &rql.EntrySchema{}
	schema.SetPath("bar")
	s.EESFTC(p, schema)
	schema.SetPath("foo")
	s.EESTTC(p, schema)
}

func (s *KindTestSuite) TestExpression_Atom() {
	expr := expression.New("kind", false, func() rql.ASTNode {
		return Kind(predicate.String())
	})

	s.MUM(expr, []interface{}{"kind", []interface{}{"glob", "foo"}})
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
}

func TestKind(t *testing.T) {
	suite.Run(t, new(KindTestSuite))
}
