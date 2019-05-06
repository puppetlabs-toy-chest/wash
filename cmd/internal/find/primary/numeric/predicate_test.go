package numeric

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/stretchr/testify/suite"
)

type PredicateTestSuite struct {
	suite.Suite
}

func (suite *PredicateTestSuite) TestParsePredicateErrors() {
	_, _, err := ParsePredicate("", ParsePositiveInt)
	suite.Regexp("empty", err)

	_, _, err = ParsePredicate("--15", ParsePositiveInt)
	suite.False(errz.IsMatchError(err))
	suite.Regexp("expected.*positive.*-15", err)

	_, _, err = ParsePredicate("foo", ParsePositiveInt)
	suite.True(errz.IsMatchError(err))
	suite.Regexp("foo.*number", err)
}

func (suite *PredicateTestSuite) TestParsePredicateValidInput() {
	type testCase struct {
		input    string
		parserID int
		trueV    int64
		falseV   int64
	}
	testCases := []testCase{
		// These test the comparison symbols "+" and "-"
		testCase{"1", 0, 1, -1},
		testCase{"+1", 0, 2, 1},
		testCase{"-1", 0, 0, 1},
		// This tests that ParsePredicate loops over all its given
		// parsers
		testCase{"1h", 1, 1 * DurationOf('h'), 2},
	}
	for _, c := range testCases {
		inputStr := func() string {
			return fmt.Sprintf("Input was '%v'", c.input)
		}
		p, parserID, err := ParsePredicate(c.input, ParsePositiveInt, ParseDuration)
		if suite.NoError(err, inputStr()) {
			suite.Equal(c.parserID, parserID)
			suite.True(p(c.trueV), inputStr())
			suite.False(p(c.falseV), inputStr())
		}
	}
}

func TestPredicate(t *testing.T) {
	suite.Run(t, new(PredicateTestSuite))
}
