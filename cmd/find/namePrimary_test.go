package cmdfind

import (
	"testing"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

type NamePrimaryTestSuite struct {
	suite.Suite
}

func (suite *NamePrimaryTestSuite) TestNamePrimaryErrors() {
	_, _, err := namePrimary.parsePredicate([]string{"-name"})
	suite.Regexp("-name: requires additional arguments", err)

	_, _, err = namePrimary.parsePredicate([]string{"-name", "[a"})
	suite.Regexp("-name: invalid glob: unexpected end of input", err)
}

func (suite *NamePrimaryTestSuite) TestNamePrimaryValidInput() {
	e1 := &apitypes.Entry{CName: "a"}
	e2 := &apitypes.Entry{CName: "b"}
	p, tokens, err := namePrimary.parsePredicate([]string{"-name", "a"})
	if suite.NoError(err) {
		suite.Equal([]string{}, tokens)
		suite.Equal(true, p(e1))
		suite.Equal(false, p(e2))
	}
}

func TestNamePrimary(t *testing.T) {
	suite.Run(t, new(NamePrimaryTestSuite))
}
