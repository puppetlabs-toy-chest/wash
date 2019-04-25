package numeric

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SignTestSuite struct {
	suite.Suite
}

func (suite *SignTestSuite) TestParsePositiveInt() {
	_, err := ParsePositiveInt("foo")
	suite.Regexp("syntax", err)

	_, err = ParsePositiveInt("-1")
	suite.Regexp("positive.*-1", err)

	n, err := ParsePositiveInt("12")
	if suite.NoError(err) {
		suite.Equal(int64(12), n)
	}
}

func (suite *SignTestSuite) TestNegate() {
	p := Negate(ParsePositiveInt)

	// Should return p's error
	_, err := p("-15")
	suite.Regexp(".*positive.*", err)

	n, err := p("15")
	if suite.NoError(err) {
		suite.Equal(int64(-15), n)
	}
}

func TestPositive(t *testing.T) {
	suite.Run(t, new(SignTestSuite))
}
