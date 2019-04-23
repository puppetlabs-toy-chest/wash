// Package meta contains all the parsing logic for the `meta` primary
package meta

import (
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type PrimaryTestSuite struct {
	suite.Suite
}

func (suite *PrimaryTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *PrimaryTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *PrimaryTestSuite) TestErrors() {
	_, _, err := Primary.Parse(toTks("-m -true"))
	suite.Regexp("-m: key sequences must begin with a '.'", err)

	_, _, err = Primary.Parse(toTks("-m ."))
	suite.Regexp("-m: expected a key sequence after '.'", err)
}

func (suite *PrimaryTestSuite) TestValidInput() {
	p, tokens, err := Primary.Parse(toTks("-m -empty -size"))
	if suite.NoError(err) {
		suite.Equal(toTks("-size"), tokens)
		e := types.Entry{}
		suite.True(p(e))
	}
}
