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
	r, err := Parse([]string{"foo", "-true"})
	if suite.NoError(err) {
		suite.Equal("foo", r.Path)
		suite.Equal(true, r.Predicate(types.Entry{}))
	}

	// Test a parse error on the expression
	r, err = Parse([]string{"foo", "-unknown"})
	suite.Regexp("unknown.*primary", err)
}

func TestParse(t *testing.T) {
	suite.Run(t, new(ParseTestSuite))
}
