package parser

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

// parsePath, parseOption, and parseExpression are tested separately.
// The tests here are meant to make sure that they integrate well
// together.

type ParseTestSuite struct {
	suite.Suite
}

func (suite *ParseTestSuite) TestParseOptionsError() {
	_, err := Parse([]string{"foo", "-unknown"})
	suite.Regexp("flag.*unknown", err)
}

func (suite *ParseTestSuite) TestParseExpressionError() {
	_, err := Parse([]string{"foo", "-true", "-a", "-blah"})
	suite.Regexp("-blah.*primary", err)
}

func (suite *ParseTestSuite) TestValidInput() {
	r, err := Parse([]string{"foo", "-depth", "-true"})
	if suite.NoError(err) {
		suite.Equal("foo", r.Path)
		expectedOpts := types.NewOptions()
		expectedOpts.MarkAsSet(types.DepthFlag)
		expectedOpts.Depth = true
		suite.Equal(expectedOpts, r.Options)
		suite.Equal(true, r.Predicate(types.Entry{}))
	}
}

func (suite *ParseTestSuite) TestPrimariesCanSetOptions() {
	// Test when an option is not set
	r, err := Parse([]string{"-meta", ".key", "-true"})
	if suite.NoError(err) {
		expectedOpts := types.NewOptions()
		expectedOpts.Maxdepth = uint(1)
		suite.Equal(expectedOpts, r.Options)
	}

	// Test when an option is set
	r, err = Parse([]string{"-maxdepth", "10", "-meta", ".key", "-true"})
	if suite.NoError(err) {
		expectedOpts := types.NewOptions()
		expectedOpts.Maxdepth = uint(10)
		expectedOpts.MarkAsSet(types.MaxdepthFlag)
		suite.Equal(expectedOpts, r.Options)
	}
}

func TestParse(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}
