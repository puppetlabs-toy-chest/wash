package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

type StringPredicateTestSuite struct {
	predicate.ParserTestSuite
}

func (s *StringPredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "expected a nonempty string", true),
		s.NPETC("-a", "-a begins with a '-'", true),
	)

	_, _, err := parseStringPredicate([]string{""})
	s.Regexp("expected a nonempty string", err)
}

func (s *StringPredicateTestSuite) TestValidInput() {
	// Test the happy cases first
	s.RunTestCases(
		s.NPTC("foo -size", "-size", "foo"),
	)

	// Now test that the predicate returns false for a non-string
	// value or if value != s
	p, _, err := parseStringPredicate(s.ToTks("foo"))
	if s.NoError(err) {
		s.False(p(200))
		s.False(p("bar"))
	}
}

func TestStringPredicate(t *testing.T) {
	s := new(StringPredicateTestSuite)
	s.Parser = predicate.GenericParser(parseStringPredicate)
	suite.Run(t, s)
}
