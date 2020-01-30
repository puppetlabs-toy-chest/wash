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
	p := Path(predicate.StringGlob(""))
	s.UMETC(p, "foo", "formatted.*'path'.*<string_predicate>", true)
	s.UMETC(p, s.A("foo", s.A("glob", "foo")), "formatted.*'path'.*<string_predicate>", true)
	s.UMETC(p, s.A("path", "foo", "bar"), "formatted.*'path'.*<string_predicate>", false)
	s.UMETC(p, s.A("path"), "missing.*string.*predicate", false)
	s.UMETC(p, s.A("path", s.A("glob", "[")), "glob", false)
	s.UMTC(p, s.A("path", s.A("glob", "foo")), Path(predicate.StringGlob("foo")))
}

func (s *PathTestSuite) TestEntryInDomain() {
	p := Path(predicate.StringGlob("foo"))
	e := rql.Entry{}
	s.EIDFTC(p, e)
	e.Path = "bar"
	s.EIDTTC(p, e)
}

func (s *PathTestSuite) TestEvalEntry() {
	p := Path(predicate.StringGlob("foo"))
	e := rql.Entry{}
	e.Path = "bar"
	s.EEFTC(p, e)
	e.Path = "foo"
	s.EETTC(p, e)
}

func (s *PathTestSuite) TestExpression_AtomAndNot() {
	expr := expression.New("path", func() rql.ASTNode {
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

	s.MUM(expr, []interface{}{"NOT", []interface{}{"path", []interface{}{"glob", "foo"}}})
	e.Path = ""
	s.EEFTC(expr, e)
	e.Path = "bar"
	s.EETTC(expr, e)
	e.Path = "foo"
	s.EEFTC(expr, e)

	s.EESTTC(expr, schema)
}

func TestPath(t *testing.T) {
	suite.Run(t, new(PathTestSuite))
}
