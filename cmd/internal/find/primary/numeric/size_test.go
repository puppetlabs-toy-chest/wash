package numeric

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SizeTestSuite struct {
	suite.Suite
}

func (suite *SizeTestSuite) TestBytesOf() {
	// Use array of bytes instead of a string to make it easier
	// to add other, longer units in the future
	testCases := []byte{
		'c',
		'k',
		'M',
		'G',
		'T',
		'P',
	}
	for _, input := range testCases {
		suite.Equal(bytesMap[input], BytesOf(input))
	}
}

func (suite *SizeTestSuite) TestSizeRegex() {
	suite.Regexp(SizeRegex, "1c")
	suite.Regexp(SizeRegex, "1k")
	suite.Regexp(SizeRegex, "1M")
	suite.Regexp(SizeRegex, "1G")
	suite.Regexp(SizeRegex, "1T")
	suite.Regexp(SizeRegex, "1P")
	suite.Regexp(SizeRegex, "12c")

	suite.NotRegexp(SizeRegex, "1")
	suite.NotRegexp(SizeRegex, "12")
	suite.NotRegexp(SizeRegex, "1f")
	suite.NotRegexp(SizeRegex, "  1c")
	suite.NotRegexp(SizeRegex, "1c  ")
}

func (suite *SizeTestSuite) TestParseSize() {
	_, err := ParseSize("")
	suite.Regexp("size.*conform", err)

	n, err := ParseSize("12k")
	if suite.NoError(err) {
		suite.Equal(12*BytesOf('k'), n)
	}
}

func TestSize(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}
