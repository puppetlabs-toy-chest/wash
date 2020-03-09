package predicate

import (
	"regexp"
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type StringGlobTestSuite struct {
	asttest.Suite
}

func (s *StringGlobTestSuite) TestMarshal() {
	s.MTC(StringGlob("foo"), s.A("glob", "foo"))
}

func (s *StringGlobTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"glob".*<glob>`, true)
	s.UMETC(s.A("foo"), `formatted.*"glob".*<glob>`, true)
	s.UMETC(s.A("glob", "foo", "bar"), `formatted.*"glob".*<glob>`, false)
	s.UMETC(s.A("glob"), `formatted.*"glob".*<glob>.*missing.*glob`, false)
	s.UMETC(s.A("glob", 1), "glob.*string", false)
	s.UMETC(s.A("glob", "["), "invalid.*glob.*[.*closing.*]", false)
}

func (s *StringGlobTestSuite) TestEvalString() {
	ast := s.A("glob", "foo")
	s.ESFTC(ast, "bar")
	s.ESTTC(ast, "foo")
}

func TestStringGlob(t *testing.T) {
	s := new(StringGlobTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return StringGlob("")
	}
	suite.Run(t, s)
}

type StringRegexTestSuite struct {
	asttest.Suite
}

func (s *StringRegexTestSuite) TestMarshal() {
	s.MTC(StringRegex(regexp.MustCompile("foo")), s.A("regex", "foo"))
}

func (s *StringRegexTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"regex".*<regex>`, true)
	s.UMETC(s.A("foo"), `formatted.*"regex".*<regex>`, true)
	s.UMETC(s.A("regex", "foo", "bar"), `formatted.*"regex".*<regex>`, false)
	s.UMETC(s.A("regex"), `formatted.*"regex".*<regex>.*missing.*regex`, false)
	s.UMETC(s.A("regex", 1), "regex.*string", false)
	s.UMETC(s.A("regex", "["), "invalid.*regex.*[.*closing.*]", false)
}

func (s *StringRegexTestSuite) TestEvalString() {
	ast := s.A("regex", "foo")
	s.ESFTC(ast, "bar")
	s.ESTTC(ast, "foo")
}

func TestStringRegex(t *testing.T) {
	s := new(StringRegexTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return StringRegex(nil)
	}
	suite.Run(t, s)
}

type StringEqualTestSuite struct {
	asttest.Suite
}

func (s *StringEqualTestSuite) TestMarshal() {
	s.MTC(StringEqual("foo"), s.A("=", "foo"))
}

func (s *StringEqualTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"=".*<str>`, true)
	s.UMETC(s.A("foo"), `formatted.*"=".*<str>`, true)
	s.UMETC(s.A("=", "foo", "bar"), `formatted.*"=".*<str>`, false)
	s.UMETC(s.A("="), `formatted.*"=".*<str>.*missing.*string`, false)
	s.UMETC(s.A("=", 1), "string", false)
}

func (s *StringEqualTestSuite) TestEvalString() {
	ast := s.A("=", "foo")
	s.ESFTC(ast, "bar")
	s.ESTTC(ast, "foo")
}

func TestStringEqual(t *testing.T) {
	s := new(StringEqualTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return StringEqual("")
	}
	suite.Run(t, s)
}

type StringTestSuite struct {
	asttest.Suite
}

func (s *StringTestSuite) TestMarshal() {
	p := String().(internal.NonterminalNode)
	p.SetMatchedNode(StringGlob("foo"))
	s.MTC(p, StringGlob("foo").Marshal())
}

func (s *StringTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"glob".*"regex".*"="`, true)
	s.UMETC(s.A("glob", "["), "invalid.*glob", false)
	s.UMETC(s.A("regex", "["), "invalid.*regex", false)
	s.UMETC(s.A("=", true), "string", false)
}

func (s *StringTestSuite) TestEvalString() {
	for _, ptype := range []string{"glob", "regex", "="} {
		ast := s.A(ptype, "foo")
		s.ESFTC(ast, "bar")
		s.ESTTC(ast, "foo")
	}
}

func (s *StringTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("string", true, func() rql.ASTNode {
			return String()
		})
	}

	for _, ptype := range []string{"glob", "regex", "="} {
		ast := s.A(ptype, "foo")
		s.ESFTC(ast, "bar")
		s.ESTTC(ast, "foo")
		s.AssertNotImplemented(
			ast,
			asttest.EntryPredicateC,
			asttest.EntrySchemaPredicateC,
			asttest.NumericPredicateC,
			asttest.TimePredicateC,
			asttest.ActionPredicateC,
		)

		notAST := s.A("NOT", ast)
		s.ESTTC(notAST, "bar")
		s.ESFTC(notAST, "foo")
	}
}

func TestString(t *testing.T) {
	s := new(StringTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return String()
	}
	suite.Run(t, s)
}

type StringValueTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *StringValueTestSuite) TestMarshal() {
	// This also tests that the StringValue* methods do the right thing
	s.MTC(StringValueGlob("foo"), s.A("string", s.A("glob", "foo")))
	s.MTC(StringValueRegex(regexp.MustCompile("foo")), s.A("string", s.A("regex", "foo")))
	s.MTC(StringValueEqual("foo"), s.A("string", s.A("=", "foo")))
}

func (s *StringValueTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"string".*NPE StringPredicate`, true)
	s.UMETC(s.A("string", "foo", "bar"), `formatted.*"string".*NPE StringPredicate`, false)
	s.UMETC(s.A("string"), `formatted.*"string".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(s.A("string", s.A()), `error.*unmarshalling.*NPE StringPredicate.*formatted.*"glob".*<glob>`, false)
}

func (s *StringValueTestSuite) TestEvalValue() {
	ast := s.A("string", s.A("glob", "foo"))
	s.EVFTC(ast, "bar", 1)
	s.EVTTC(ast, "foo")
}

func (s *StringValueTestSuite) TestEvalValueSchema() {
	ast := s.A("string", s.A("glob", "foo"))
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("string")...)
}

func (s *StringValueTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("string", true, func() rql.ASTNode {
			return StringValue(String())
		})
	}

	ast := s.A("string", s.A("glob", "foo"))
	s.EVFTC(ast, "bar")
	s.EVTTC(ast, "foo")
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("string")...)
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.EVTTC(notAST, "bar")
	s.EVFTC(notAST, "foo")
	s.EVSTTC(notAST, s.VS("object", "array", "string")...)
}

func TestStringValue(t *testing.T) {
	s := new(StringValueTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return StringValue(String())
	}
	suite.Run(t, s)
}
