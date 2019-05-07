package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type StringPredicateTestSuite struct {
	parsertest.Suite
}

func (s *StringPredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "expected a nonempty string", true),
	)

	_, _, err := parseStringPredicate([]string{""})
	s.Regexp("expected a nonempty string", err)
}

func (s *StringPredicateTestSuite) TestValidInput() {
	s.RunTestCases(
		s.NPTC("foo -size", "-size", "foo"),
		s.NPNTC("foo -size", "-size", "bar"),
	)
}

func (s *StringPredicateTestSuite) TestStringP_NotAString() {
	sp := stringP(func(s string) bool {
		return s == "f"
	})

	s.False(sp.IsSatisfiedBy('f'))
	s.False(sp.Negate().IsSatisfiedBy('g'))
}

func (s *StringPredicateTestSuite) TestStringP() {
	sp := stringP(func(s string) bool {
		return s == "f"
	})

	s.True(sp.IsSatisfiedBy("f"))
	
	// Test negation
	s.False(sp.Negate().IsSatisfiedBy("f"))
	s.True(sp.Negate().IsSatisfiedBy("g"))
}

func TestStringPredicate(t *testing.T) {
	s := new(StringPredicateTestSuite)
	s.Parser = predicate.ToParser(parseStringPredicate)
	suite.Run(t, s)
}
