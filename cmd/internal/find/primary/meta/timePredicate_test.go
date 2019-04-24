package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type TimePredicateTestSuite struct {
	ParserTestSuite
}

func (suite *TimePredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *TimePredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *TimePredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", `expected a \+, -, or a digit`, true),
		nPETC("200", "expected a duration", true),
		nPETC("+{", ".*closing.*}", false),
	)
}

func (suite *TimePredicateTestSuite) TestValidInputTrueValues() {
	// Test the happy cases first
	suite.runTestCases(
		nPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h'))),
		nPTC("-2h -size", "-size", addTST(-1*numeric.DurationOf('h'))),
		nPTC("+{2h} -size", "-size", addTST(3*numeric.DurationOf('h'))),
		nPTC("-{2h} -size", "-size", addTST(1*numeric.DurationOf('h'))),
		// Test a stringified time to ensure that munge.ToTime's called
		nPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h')).String()),
	)
}

func (suite *TimePredicateTestSuite) TestValidInputFalseValues() {
	// TODO: May be worth adding support for false values to the
	// parserTestCase class if testing lots of them becomes common
	// enough.
	type negativeTestCase struct {
		input string
		falseV interface{}
	}
	cases := []negativeTestCase{
		negativeTestCase{"+2h", "not_a_valid_time_value"},
		negativeTestCase{"+2h", addTST(-1*numeric.DurationOf('h'))},
		negativeTestCase{"-2h", addTST(-3*numeric.DurationOf('h'))},
		negativeTestCase{"+{2h}", addTST(1*numeric.DurationOf('h'))},
		negativeTestCase{"-{2h}", addTST(3*numeric.DurationOf('h'))},
	}
	for _, c := range cases {
		p, _, err := parseTimePredicate(toTks(c.input))
		if suite.NoError(err, "Input: %v", c.input) {
			suite.False(p(c.falseV), "Input: %v", c.input)
		}
	}
}

// addTST => addToStartTime. Saves some typing. Note that v
// is an int64 duration.
func addTST(v int64) time.Time {
	return params.StartTime.Add(time.Duration(v))
}

func TestTimePredicate(t *testing.T) {
	s := new(TimePredicateTestSuite)
	s.parser = parseTimePredicate
	suite.Run(t, s)
}
