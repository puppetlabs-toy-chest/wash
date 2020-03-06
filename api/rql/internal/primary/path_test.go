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
	n := Path(predicate.StringGlob(""))
	s.UMETC(n, "foo", `path.*formatted.*"path".*NPE StringPredicate`, true)
	s.UMETC(n, s.A("foo", s.A("glob", "foo")), `path.*formatted.*"path".*NPE StringPredicate`, true)
	s.UMETC(n, s.A("path", "foo", "bar"), `path.*formatted.*"path".*NPE StringPredicate`, false)
	s.UMETC(n, s.A("path"), `path.*formatted.*"path".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(n, s.A("path", s.A("glob", "[")), "path.*NPE StringPredicate.*glob", false)
	s.UMTC(n, s.A("path", s.A("glob", "foo")), Path(predicate.StringGlob("foo")))
}

func (s *PathTestSuite) TestEvalEntry() {
	p := Path(predicate.StringGlob("foo"))
	e := rql.Entry{}
	e.Path = "bar"
	s.EEFTC(p, e)
	e.Path = "foo"
	s.EETTC(p, e)
}

func (s *PathTestSuite) TestExpression_Atom() {
	expr := expression.New("path", false, func() rql.ASTNode {
		return Path(predicate.String())
	})

	s.MUM(expr, []interface{}{"path", []interface{}{"glob", "foo"}})
	e := rql.Entry{}
	e.Path = ""
	s.EEFTC(expr, e)
	e.Path = "bar"
	s.EEFTC(expr, e)
	e.Path = "foo"
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
}

func TestPath(t *testing.T) {
	suite.Run(t, new(PathTestSuite))
}