package numeric

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
	"github.com/stretchr/testify/suite"
)

type PositiveTestSuite struct {
	suite.Suite
}

func (suite *PositiveTestSuite) TestParsePositiveInt() {
	_, err := ParsePositiveInt("foo")
	suite.Regexp("syntax", err)

	_, err = ParsePositiveInt("-1")
	suite.Regexp("positive.*-1", err)

	n, err := ParsePositiveInt("12")
	if suite.NoError(err) {
		suite.Equal(int64(12), n)
	}
}

func (suite *PositiveTestSuite) TestNegateErrors() {
	p := Negate(ParsePositiveInt)

	_, err := p("")
	suite.Regexp("expected a number", err)

	_, err = p("f")
	suite.Regexp(`expected an opening '{'`, err)

	_, err = p("}")
	suite.False(errz.IsMatchError(err))
	suite.Regexp(`expected an opening '{'`, err)

	_, err = p("{")
	suite.Regexp(`expected a closing '}'`, err)

	_, err = p("{a}")
	suite.Regexp("expected a number inside '{}', got: a", err)

	// Returns the underlying parser's error
	_, err = p("{-15}")
	suite.Regexp("positive.*number", err)
}

func (suite *PositiveTestSuite) TestNegateValidInput() {
	p := Negate(ParsePositiveInt)
	n, err := p("{15}")
	if suite.NoError(err) {
		suite.Equal(int64(-15), n)
	}
}

func TestPositive(t *testing.T) {
	suite.Run(t, new(PositiveTestSuite))
}
