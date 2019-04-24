package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StringPredicateTestSuite struct {
	ParserTestSuite
}

func (suite *StringPredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", "expected a nonempty string", true),
		nPETC("-a", "-a begins with a '-'", true),
	)

	_, _, err := parseStringPredicate([]string{""})
	suite.Regexp("expected a nonempty string", err)
}

func (suite *StringPredicateTestSuite) TestValidInput() {
	// Test the happy cases first
	suite.runTestCases(
		nPTC("foo -size", "-size", "foo"),
	)

	// Now test that the predicate returns false for a non-string
	// value or if value != s
	p, _, err := parseStringPredicate(toTks("foo"))
	if suite.NoError(err) {
		suite.False(p(200))
		suite.False(p("bar"))
	}
}

func TestStringPredicate(t *testing.T) {
	s := new(StringPredicateTestSuite)
	s.parser = parseStringPredicate
	suite.Run(t, s)
}
