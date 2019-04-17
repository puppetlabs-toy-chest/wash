package find

import (
	"fmt"
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type TimeAttrPrimaryTestSuite struct {
	suite.Suite
}

func (suite *TimeAttrPrimaryTestSuite) SetupTest() {
	startTime = time.Now()
}

func (suite *TimeAttrPrimaryTestSuite) TeardownTest() {
	startTime = time.Time{}
}

func (suite *TimeAttrPrimaryTestSuite) TestDurationOf() {
	// Use array of bytes instead of a string to make it easier
	// to add other, longer units in the future
	testCases := []byte{
		's',
		'm',
		'h',
		'd',
		'w',
	}
	for _, input := range testCases {
		suite.Equal(durationsMap[input], durationOf(input))
	}
}

func (suite *TimeAttrPrimaryTestSuite) TestGetTimeAttrValue() {
	e := types.Entry{}

	// Test ctime
	_, ok := getTimeAttrValue("ctime", e)
	suite.False(ok)
	expected := time.Now()
	e.Attributes.SetCtime(expected)
	actual, ok := getTimeAttrValue("ctime", e)
	if suite.True(ok) {
		suite.Equal(expected, actual)
	}

	// Test mtime
	_, ok = getTimeAttrValue("mtime", e)
	suite.False(ok)
	expected = time.Now()
	e.Attributes.SetMtime(expected)
	actual, ok = getTimeAttrValue("mtime", e)
	if suite.True(ok) {
		suite.Equal(expected, actual)
	}

	// Test atime
	_, ok = getTimeAttrValue("atime", e)
	suite.False(ok)
	expected = time.Now()
	e.Attributes.SetAtime(expected)
	actual, ok = getTimeAttrValue("atime", e)
	if suite.True(ok) {
		suite.Equal(expected, actual)
	}
}

func (suite *TimeAttrPrimaryTestSuite) TestParseDuration() {
	type testCase struct {
		input             string
		expectedDuration  time.Duration
		expectedRoundDiff bool
	}
	testCases := []testCase{
		testCase{"1", 1 * durationOf('d'), true},
		testCase{"2", 2 * durationOf('d'), true},
		testCase{"1w1d1h1m1s", 1*durationOf('w') + 1*durationOf('d') + 1*durationOf('h') + 1*durationOf('m') + 1*durationOf('s'), false},
		testCase{"2w2d2h2m2s", 2*durationOf('w') + 2*durationOf('d') + 2*durationOf('h') + 2*durationOf('m') + 2*durationOf('s'), false},
	}
	for _, testCase := range testCases {
		actualDuration, actualRoundDiff := parseDuration(testCase.input)
		suite.Equal(testCase.expectedDuration, actualDuration)
		suite.Equal(testCase.expectedRoundDiff, actualRoundDiff)
	}
}

// These tests use the ctimePrimary as the representative test case

func (suite *TimeAttrPrimaryTestSuite) TestTimeAttrPrimaryInsufficientArgsError() {
	_, _, err := ctimePrimary.parse([]string{"-ctime"})
	suite.Equal("-ctime: requires additional arguments", err.Error())
}

func (suite *TimeAttrPrimaryTestSuite) TestTimeAttrPrimaryIllegalTimeValueError() {
	illegalValues := []string{
		"foo",
		"+",
		"+++++1",
		"1hr",
		"+1hr",
		"++++++1hr",
		"1h30min",
		"+1h30min",
	}
	for _, v := range illegalValues {
		_, _, err := ctimePrimary.parse([]string{"-ctime", v})
		msg := fmt.Sprintf("-ctime: %v: illegal time value", v)
		suite.Equal(msg, err.Error())
	}
}

func (suite *TimeAttrPrimaryTestSuite) TestTimeAttrPrimaryValidInput() {
	type testCase struct {
		input string
		// trueCtime/falseCtime represent ctime durations that, when subtracted
		// from startTime, satisfy/unsatisfy the predicate, respectively.
		trueCtime  time.Duration
		falseCtime time.Duration
	}
	testCases := []testCase{
		// We set trueCtime to 1.5 days in order to test roundDiff
		testCase{"2", 1*durationOf('d') + 12*time.Hour, 1 * durationOf('d')},
		// +1 means p will return true if diff > 1 day
		testCase{"+1", 2 * durationOf('d'), 0 * durationOf('d')},
		// -2 means p will return true if diff < 2 days
		testCase{"-2", 1 * durationOf('d'), 3 * durationOf('d')},
		// Units like "1h30m" aren't really useful unless they're used with the
		// +/- modifiers, but we'd still like to test an exact comparison since it
		// is technically supported.
		testCase{"1h", 1 * durationOf('h'), 1 * durationOf('m')},
		testCase{"+1h", 2 * durationOf('h'), 1 * durationOf('m')},
		testCase{"-1h", 1 * durationOf('m'), 1 * durationOf('h')},
	}
	for _, testCase := range testCases {
		inputStr := func() string {
			return fmt.Sprintf("Input was '%v'", testCase.input)
		}
		p, tokens, err := ctimePrimary.parse([]string{"-ctime", testCase.input})
		if suite.NoError(err, inputStr()) {
			suite.Equal([]string{}, tokens)
			e := types.Entry{}
			// Ensure p(e) is always false for an entry that doesn't have a ctime attribute
			suite.False(p(e), inputStr())

			e.Attributes.SetCtime(startTime.Add(-testCase.trueCtime))
			suite.True(p(e), inputStr())

			e.Attributes.SetCtime(startTime.Add(-testCase.falseCtime))
			suite.False(p(e), inputStr())
		}
	}
}

func TestTimeAttrPrimary(t *testing.T) {
	suite.Run(t, new(TimeAttrPrimaryTestSuite))
}
