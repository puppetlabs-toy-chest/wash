package numeric

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DurationTestSuite struct {
	suite.Suite
}

func (suite *DurationTestSuite) TestDurationOf() {
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
		suite.Equal(int64(durationsMap[input]), DurationOf(input))
	}
}

func (suite *DurationTestSuite) TestDurationRegex() {
	suite.Regexp(DurationRegex, "1s")
	suite.Regexp(DurationRegex, "1m")
	suite.Regexp(DurationRegex, "1h")
	suite.Regexp(DurationRegex, "1d")
	suite.Regexp(DurationRegex, "1w")
	suite.Regexp(DurationRegex, "12s")
	suite.Regexp(DurationRegex, "1h30m")

	suite.NotRegexp(DurationRegex, "1")
	suite.NotRegexp(DurationRegex, "12")
	suite.NotRegexp(DurationRegex, "1f")
	suite.NotRegexp(DurationRegex, "  1s")
	suite.NotRegexp(DurationRegex, "1s  ")
}

func (suite *DurationTestSuite) TestParseDuration() {
	_, err := ParseDuration("")
	suite.Regexp("duration.*conform", err)

	n, err := ParseDuration("1h30m")
	if suite.NoError(err) {
		suite.Equal(1*DurationOf('h')+30*DurationOf('m'), n)
	}
}

func TestDuration(t *testing.T) {
	suite.Run(t, new(DurationTestSuite))
}
