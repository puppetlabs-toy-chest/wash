package primary

import (
	"fmt"
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type TimeAttrPrimaryTestSuite struct {
	suite.Suite
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
		trueCtime  int64
		falseCtime int64
	}
	testCases := []testCase{
		// We set trueCtime to 1.5 days in order to test roundDiff
		testCase{"2", 1*numeric.DurationOf('d') + 12*numeric.DurationOf('h'), 1 * numeric.DurationOf('d')},
		// +1 means p will return true if diff > 1 day
		testCase{"+1", 2 * numeric.DurationOf('d'), 0 * numeric.DurationOf('d')},
		// -2 means p will return true if diff < 2 days
		testCase{"-2", 1 * numeric.DurationOf('d'), 3 * numeric.DurationOf('d')},
		// time.Time has nanosecond precision so units like "1h30m" aren't really
		// useful unless they're used with the +/- modifiers.
		testCase{"+1h", 2 * numeric.DurationOf('h'), 1 * numeric.DurationOf('m')},
		testCase{"-1h", 1 * numeric.DurationOf('m'), 1 * numeric.DurationOf('h')},
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

			e.Attributes.SetCtime(params.StartTime.Add(time.Duration(-testCase.trueCtime)))
			suite.True(p(e), inputStr())

			e.Attributes.SetCtime(params.StartTime.Add(time.Duration(-testCase.falseCtime)))
			suite.False(p(e), inputStr())
		}
	}
}

func TestTimeAttrPrimary(t *testing.T) {
	suite.Run(t, new(TimeAttrPrimaryTestSuite))
}
