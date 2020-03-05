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

type StringTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *StringTestSuite) TestGlob_Marshal() {
	s.MTC(StringGlob("foo"), s.A("glob", "foo"))
}

func (s *StringTestSuite) TestGlob_Unmarshal() {
	g := StringGlob("")
	s.UMETC(g, "foo", `formatted.*"glob".*<glob>`, true)
	s.UMETC(g, s.A("foo"), `formatted.*"glob".*<glob>`, true)
	s.UMETC(g, s.A("glob", "foo", "bar"), `formatted.*"glob".*<glob>`, false)
	s.UMETC(g, s.A("glob"), `formatted.*"glob".*<glob>.*missing.*glob`, false)
	s.UMETC(g, s.A("glob", 1), "glob.*string", false)
	s.UMETC(g, s.A("glob", "["), "invalid.*glob.*[.*closing.*]", false)
	s.UMTC(g, s.A("glob", "foo"), StringGlob("foo"))
}

func (s *StringTestSuite) TestGlob_EvalString() {
	g := StringGlob("foo")
	s.ESFTC(g, "bar")
	s.ESTTC(g, "foo")
}

func (s *StringTestSuite) TestRegex_Marshal() {
	s.MTC(StringRegex(regexp.MustCompile("foo")), s.A("regex", "foo"))
}

func (s *StringTestSuite) TestRegex_Unmarshal() {
	r := StringRegex(nil)
	s.UMETC(r, "foo", `formatted.*"regex".*<regex>`, true)
	s.UMETC(r, s.A("foo"), `formatted.*"regex".*<regex>`, true)
	s.UMETC(r, s.A("regex", "foo", "bar"), `formatted.*"regex".*<regex>`, false)
	s.UMETC(r, s.A("regex"), `formatted.*"regex".*<regex>.*missing.*regex`, false)
	s.UMETC(r, s.A("regex", 1), "regex.*string", false)
	s.UMETC(r, s.A("regex", "["), "invalid.*regex.*[.*closing.*]", false)
	s.UMTC(r, s.A("regex", "foo"), StringRegex(regexp.MustCompile("foo")))
}

func (s *StringTestSuite) TestRegex_EvalString() {
	r := StringRegex(regexp.MustCompile("foo"))
	s.ESFTC(r, "bar")
	s.ESTTC(r, "foo")
}

func (s *StringTestSuite) TestEqual_Marshal() {
	s.MTC(StringEqual("foo"), s.A("=", "foo"))
}

func (s *StringTestSuite) TestEqual_Unmarshal() {
	e := StringEqual("")
	s.UMETC(e, "foo", `formatted.*"=".*<str>`, true)
	s.UMETC(e, s.A("foo"), `formatted.*"=".*<str>`, true)
	s.UMETC(e, s.A("=", "foo", "bar"), `formatted.*"=".*<str>`, false)
	s.UMETC(e, s.A("="), `formatted.*"=".*<str>.*missing.*string`, false)
	s.UMETC(e, s.A("=", 1), "string", false)
	s.UMTC(e, s.A("=", "foo"), StringEqual("foo"))
}

func (s *StringTestSuite) TestEqual_EvalString() {
	e := StringEqual("foo")
	s.ESFTC(e, "bar")
	s.ESTTC(e, "foo")
}

func (s *StringTestSuite) TestString_Marshal() {
	p := String().(internal.NonterminalNode)
	p.SetMatchedNode(StringGlob("foo"))
	s.MTC(p, StringGlob("foo").Marshal())
}

func (s *StringTestSuite) TestString_Unmarshal() {
	p := String()
	s.UMETC(p, "foo", `formatted.*"glob".*"regex".*"="`, true)
	s.UMETC(p, s.A("glob", "["), "invalid.*glob", false)
	s.UMETC(p, s.A("regex", "["), "invalid.*regex", false)
	s.UMETC(p, s.A("=", true), "string", false)

	s.UMTC(p, s.A("glob", "foo"), StringGlob("foo"))
	s.UMTC(p, s.A("regex", "foo"), StringRegex(regexp.MustCompile("foo")))
	s.UMTC(p, s.A("=", "foo"), StringEqual("foo"))
}

func (s *StringTestSuite) TestString_EvalString() {
	p := String().(internal.NonterminalNode)
	p.SetMatchedNode(StringGlob("foo"))
	s.ESFTC(p, "bar")
	s.ESTTC(p, "foo")
}

func (s *StringTestSuite) TestString_Expression_AtomAndNot() {
	expr := expression.New("string", true, func() rql.ASTNode {
		return String()
	})

	for _, ptype := range []string{"glob", "regex", "="} {
		s.MUM(expr, []interface{}{ptype, "foo"})
		s.ESFTC(expr, "bar")
		s.ESTTC(expr, "foo")
		s.AssertNotImplemented(
			expr,
			asttest.EntryPredicateC,
			asttest.EntrySchemaPredicateC,
			asttest.NumericPredicateC,
			asttest.TimePredicateC,
			asttest.ActionPredicateC,
		)

		s.MUM(expr, []interface{}{"NOT", []interface{}{ptype, "foo"}})
		s.ESTTC(expr, "bar")
		s.ESFTC(expr, "foo")
	}
}

func (s *StringTestSuite) TestStringValue_Marshal() {
	// This also tests that the StringValue* methods do the right thing
	s.MTC(StringValueGlob("foo"), s.A("string", s.A("glob", "foo")))
	s.MTC(StringValueRegex(regexp.MustCompile("foo")), s.A("string", s.A("regex", "foo")))
	s.MTC(StringValueEqual("foo"), s.A("string", s.A("=", "foo")))
}

func (s *StringTestSuite) TestStringValue_Unmarshal() {
	g := StringValueGlob("")
	s.UMETC(g, "foo", `formatted.*"string".*NPE StringPredicate`, true)
	s.UMETC(g, s.A("string", "foo", "bar"), `formatted.*"string".*NPE StringPredicate`, false)
	s.UMETC(g, s.A("string"), `formatted.*"string".*NPE StringPredicate.*missing.*NPE StringPredicate`, false)
	s.UMETC(g, s.A("string", s.A()), `error.*unmarshalling.*NPE StringPredicate.*formatted.*"glob".*<glob>`, false)
	s.UMTC(g, s.A("string", s.A("glob", "foo")), StringValueGlob("foo"))
}

func (s *StringTestSuite) TestStringValue_EvalValue() {
	g := StringValueGlob("foo")
	s.EVFTC(g, "bar", 1)
	s.EVTTC(g, "foo")
}

func (s NumericTestSuite) TestStringValue_EvalValueSchema() {
	g := StringValueGlob("foo")
	s.EVSFTC(g, s.VS("object", "array")...)
	s.EVSTTC(g, s.VS("string")...)
}

func (s *StringTestSuite) TestStringValue_AtomAndNot() {
	expr := expression.New("string", true, func() rql.ASTNode {
		return StringValue(String())
	})

	s.MUM(expr, []interface{}{"string", []interface{}{"glob", "foo"}})
	s.EVFTC(expr, "bar")
	s.EVTTC(expr, "foo")
	s.EVSFTC(expr, s.VS("object", "array")...)
	s.EVSTTC(expr, s.VS("string")...)
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"string", []interface{}{"glob", "foo"}}})
	s.EVTTC(expr, "bar")
	s.EVFTC(expr, "foo")
	s.EVSTTC(expr, s.VS("object", "array", "string")...)
}

func TestString(t *testing.T) {
	suite.Run(t, new(StringTestSuite))
}
