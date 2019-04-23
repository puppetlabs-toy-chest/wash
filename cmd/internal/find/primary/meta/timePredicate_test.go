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

func (suite *TimePredicateTestSuite) TestValidInput() {
	// Test the happy cases first
	suite.runTestCases(
		nPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h'))),
		nPTC("-2h -size", "-size", addTST(-1*numeric.DurationOf('h'))),
		nPTC("+{2h} -size", "-size", addTST(3*numeric.DurationOf('h'))),
		nPTC("-{2h} -size", "-size", addTST(1*numeric.DurationOf('h'))),
		// Test a stringified time to ensure that munge.ToTime's called
		nPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h')).String()),
	)

	// Now test that the predicate returns false for a non-time
	// value
	p, _, err := parseTimePredicate(toTks("+2h"))
	if suite.NoError(err) {
		suite.False(p("not_a_time"))
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
