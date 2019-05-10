package primary

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type ActionPrimaryTestSuite struct {
	primaryTestSuite
}

func (s *ActionPrimaryTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "requires additional arguments"),
		s.NPETC("foo", "foo is an invalid action. Valid actions are.*list"),
	)
}

func (s *ActionPrimaryTestSuite) TestValidInput() {
	s.RTC("list", "", []string{"list"}, []string{"exec"})
	// Test multiple supported actions
	s.RTC("list", "", []string{"read", "stream", "list"}, []string{"read", "stream"})
}

func TestActionPrimary(t *testing.T) {
	s := new(ActionPrimaryTestSuite)
	s.Parser = actionPrimary
	s.ConstructEntry = func(v interface{}) types.Entry {
		e := types.Entry{}
		e.Actions = v.([]string)
		return e
	}
	suite.Run(t, s)
}
