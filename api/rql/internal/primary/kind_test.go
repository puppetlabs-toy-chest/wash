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
	s.UMETC("foo", `kind.*formatted.*"kind".*NPE StringPredicate`, true)
	s.UMETC(s.A("foo", s.A("glob", "foo")), `kind.*formatted.*"kind".*NPE StringPredicate`, true)
	s.UMETC(s.A("kind", "foo", "bar"), `kind.*formatted.*"kind".*NPE StringPredicate`, false)
	s.UMETC(s.A("kind"), `kind.*formatted.*"kind".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(s.A("kind", s.A("glob", "[")), "kind.*NPE StringPredicate.*glob", false)
}
func (s *KindTestSuite) TestEvalEntrySchema() {
	ast := s.A("kind", s.A("glob", "foo"))
	schema := &rql.EntrySchema{}
	schema.SetPath("bar")
	s.EESFTC(ast, schema)
	schema.SetPath("foo")
	s.EESTTC(ast, schema)
}

func (s *KindTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("kind", false, func() rql.ASTNode {
			return Kind(predicate.String())
		})
	}

	ast := s.A("kind", s.A("glob", "foo"))
	e := rql.Entry{}
	s.EEFTC(ast, e)
	e.Schema = &rql.EntrySchema{}
	e.Schema.SetPath("bar")
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
	schema.SetPath("")
	s.EESFTC(ast, schema)
	schema.SetPath("bar")
	s.EESFTC(ast, schema)
	schema.SetPath("foo")
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

func TestKind(t *testing.T) {
	s := new(KindTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Kind(predicate.String())
	}
	suite.Run(t, s)
}
