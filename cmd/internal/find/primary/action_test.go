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
	_, _, err := actionPrimary.Parse([]string{"-action"})
	suite.Equal("-action: requires additional arguments", err.Error())
}

func (suite *ActionPrimaryTestSuite) TestActionPrimarySyntaxErrors() {
	_, _, err := actionPrimary.Parse([]string{"-action", ","})
	suite.Regexp("expected an action before ','", err)

	_, _, err = actionPrimary.Parse([]string{"-action", ",list"})
	suite.Regexp("expected an action before ','", err)

	_, _, err = actionPrimary.Parse([]string{"-action", ",,list"})
	suite.Regexp("expected an action before ','", err)

	_, _, err = actionPrimary.Parse([]string{"-action", "list,"})
	suite.Regexp("expected an action after ','", err)

	_, _, err = actionPrimary.Parse([]string{"-action", "list,,"})
	suite.Regexp("expected an action after ','", err)
}

func (suite *ActionPrimaryTestSuite) TestActionPrimaryInvalidActionError() {
	_, _, err := actionPrimary.Parse([]string{"-action", "foo"})
	suite.Regexp("foo is an invalid action. Valid actions are.*list", err)
	
	// Test a comma-separated list
	_, _, err = actionPrimary.Parse([]string{"-action", "list,exec,foo,read"})
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
		// Test comma-separated list
		testCase{"list,exec", []string{"exec"}, []string{"read"}},
		// Test multiple supported actions
		testCase{"list", []string{"read","stream","list"}, []string{"read", "stream"}},
		testCase{"list,exec", []string{"read","stream","list"}, []string{"read", "stream"}},
	}
	for _, testCase := range testCases {
		inputStr := func() string {
			return fmt.Sprintf("Input was '%v'", testCase.input)
		}
		p, tokens, err := actionPrimary.Parse([]string{"-action", testCase.input})
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
