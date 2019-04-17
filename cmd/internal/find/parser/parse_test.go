package parser

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type ParseTestSuite struct {
	suite.Suite
}

// parsePath + parseExpression are tested separately, so TestParse()
// can just make sure that they integrate well together.
func (suite *ParseTestSuite) TestParse() {
	// Happy case
	r, err := Parse([]string{"foo", "-depth", "-true"})
	if suite.NoError(err) {
		suite.Equal("foo", r.Path)
		expectedOpts := types.NewOptions()
		expectedOpts.Depth = true
		suite.Equal(expectedOpts, r.Options)
		suite.Equal(true, r.Predicate(types.Entry{}))
	}

	// Test a parse error on the options
	r, err = Parse([]string{"foo", "-unknown"})
	suite.Regexp("flag.*unknown", err)

	// Test a parse error on the expression
	r, err = Parse([]string{"foo", "-true", "-a", "-blah"})
	suite.Regexp("-blah.*primary", err)
}

func TestParse(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}
