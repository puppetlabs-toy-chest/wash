package numeric

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
	"github.com/stretchr/testify/suite"
)

type BracketTestSuite struct {
	suite.Suite
}

func (suite *BracketTestSuite) TestBracketErrors() {
	p := Bracket(ParsePositiveInt)

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

func (suite *BracketTestSuite) TestBracketValidInput() {
	p := Bracket(ParsePositiveInt)
	n, err := p("{15}")
	if suite.NoError(err) {
		suite.Equal(int64(15), n)
	}
}

func TestBracket(t *testing.T) {
	suite.Run(t, new(BracketTestSuite))
}
