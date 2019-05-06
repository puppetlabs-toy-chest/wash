package primary

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type ActionPrimaryTestSuite struct {
	suite.Suite
}

func (suite *ActionPrimaryTestSuite) TestActionPrimaryInsufficientArgsError() {
	_, _, err := actionPrimary.parse([]string{"-action"})
	suite.Equal("-action: requires additional arguments", err.Error())
}

func (suite *ActionPrimaryTestSuite) TestActionPrimaryInvalidActionError() {
	_, _, err := actionPrimary.parse([]string{"-action", "foo"})
	suite.Regexp("foo is an invalid action. Valid actions are.*list", err)
}

func (suite *ActionPrimaryTestSuite) TestActionPrimaryValidInput() {
	type testCase struct {
		input string
		// trueActions/falseActions represent entry actions that satisfy/unsatisfy
		// the predicate, respectively.
		trueActions  []string
		falseActions []string
	}
	testCases := []testCase{
		testCase{"list", []string{"list"}, []string{"exec"}},
		// Test multiple supported actions
		testCase{"list", []string{"read","stream","list"}, []string{"read", "stream"}},
	}
	for _, testCase := range testCases {
		inputStr := func() string {
			return fmt.Sprintf("Input was '%v'", testCase.input)
		}
		p, tokens, err := actionPrimary.parse([]string{"-action", testCase.input})
		if suite.NoError(err, inputStr()) {
			suite.Equal([]string{}, tokens)
			e := types.Entry{}
			
			e.Actions = testCase.trueActions
			suite.True(p(e), inputStr())

			e.Actions = testCase.falseActions
			suite.False(p(e), inputStr())
		}
	}
}

func TestActionPrimary(t *testing.T) {
	suite.Run(t, new(ActionPrimaryTestSuite))
}
