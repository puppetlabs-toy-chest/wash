package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type PathTestSuite struct {
	asttest.Suite
}

func (s *PathTestSuite) TestMarshal() {
	s.MTC(Path(predicate.StringGlob("foo")), s.A("path", s.A("glob", "foo")))
}

func (s *PathTestSuite) TestUnmarshal() {
	s.UMETC("foo", `path.*formatted.*"path".*NPE StringPredicate`, true)
	s.UMETC(s.A("foo", s.A("glob", "foo")), `path.*formatted.*"path".*NPE StringPredicate`, true)
	s.UMETC(s.A("path", "foo", "bar"), `path.*formatted.*"path".*NPE StringPredicate`, false)
	s.UMETC(s.A("path"), `path.*formatted.*"path".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(s.A("path", s.A("glob", "[")), "path.*NPE StringPredicate.*glob", false)
}

func (s *PathTestSuite) TestEvalEntry() {
	ast := s.A("path", s.A("glob", "foo"))
	e := rql.Entry{}
	e.Path = "bar"
	s.EEFTC(ast, e)
	e.Path = "foo"
	s.EETTC(ast, e)
}

func (s *PathTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("path", false, func() rql.ASTNode {
			return Path(predicate.String())
		})
	}

	ast := s.A("path", s.A("glob", "foo"))
	e := rql.Entry{}
	e.Path = ""
	s.EEFTC(ast, e)
	e.Path = "bar"
	s.EEFTC(ast, e)
	e.Path = "foo"
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

func TestPath(t *testing.T) {
	s := new(PathTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Path(predicate.String())
	}
	suite.Run(t, s)
}
