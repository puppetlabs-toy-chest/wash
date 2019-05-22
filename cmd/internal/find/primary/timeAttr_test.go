package primary

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type TimeAttrPrimaryTestSuite struct {
	primaryTestSuite
}

func (s *TimeAttrPrimaryTestSuite) TestGetTimeAttrValue() {
	e := types.Entry{}

	// Test ctime
	_, ok := getTimeAttrValue("ctime", e)
	s.False(ok)
	expected := time.Now()
	e.Attributes.SetCtime(expected)
	actual, ok := getTimeAttrValue("ctime", e)
	if s.True(ok) {
		s.Equal(expected, actual)
	}

	// Test mtime
	_, ok = getTimeAttrValue("mtime", e)
	s.False(ok)
	expected = time.Now()
	e.Attributes.SetMtime(expected)
	actual, ok = getTimeAttrValue("mtime", e)
	if s.True(ok) {
		s.Equal(expected, actual)
	}

	// Test atime
	_, ok = getTimeAttrValue("atime", e)
	s.False(ok)
	expected = time.Now()
	e.Attributes.SetAtime(expected)
	actual, ok = getTimeAttrValue("atime", e)
	if s.True(ok) {
		s.Equal(expected, actual)
	}
}

// These tests use the ctimePrimary as the representative test case

func (s *TimeAttrPrimaryTestSuite) TestErrors() {
	// RIVTC => RunIllegalValueTestCase
	RIVTC := func(v string) {
		s.RETC(v, fmt.Sprintf("%v: illegal time value", regexp.QuoteMeta(v)))
	}
	s.RETC("", "requires additional arguments")
	RIVTC("foo")
	RIVTC("+")
	RIVTC("+++++1")
	RIVTC("1hr")
	RIVTC("+1hr")
	RIVTC("++++++1hr")
	RIVTC("1h30min")
	RIVTC("+1h30min")
}

func (s *TimeAttrPrimaryTestSuite) TestValidInput() {
	// We set trueCtime to 1.5 days in order to test roundDiff
	s.RTC("2", "", 1*numeric.DurationOf('d') + 12*numeric.DurationOf('h'), 1 * numeric.DurationOf('d'))
	// +1 means p will return true if diff > 1 day
	s.RTC("+1", "", 2 * numeric.DurationOf('d'), 0 * numeric.DurationOf('d'))
	// -2 means p will return true if diff < 2 days
	s.RTC("-2", "", 1 * numeric.DurationOf('d'), 3 * numeric.DurationOf('d'))
	// time.Time has nanosecond precision so units like "1h30m" aren't really
	// useful unless they're used with the +/- modifiers.
	s.RTC("+1h", "", 2 * numeric.DurationOf('h'), 1 * numeric.DurationOf('m'))
	s.RTC("-1h", "", 1 * numeric.DurationOf('m'), 1 * numeric.DurationOf('h'))
}

func TestTimeAttrPrimary(t *testing.T) {
	s := new(TimeAttrPrimaryTestSuite)
	s.Parser = Ctime
	s.ConstructEntry = func(v interface{}) types.Entry {
		e := types.Entry{}
		d := time.Duration(v.(int64))
		e.Attributes.SetCtime(params.ReferenceTime.Add(-d))
		return e
	}
	suite.Run(t, s)
}
