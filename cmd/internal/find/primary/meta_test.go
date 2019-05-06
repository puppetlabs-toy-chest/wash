package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/stretchr/testify/suite"
)

type MetaPrimaryTestSuite struct {
	suite.Suite
}

func (suite *MetaPrimaryTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *MetaPrimaryTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *MetaPrimaryTestSuite) TestMetaPrimaryErrors() {
	_, _, err := metaPrimary.parse([]string{"-m", "-true"})
	suite.Regexp("-m: key sequences must begin with a '.'", err)

	_, _, err = metaPrimary.parse([]string{"-m", "."})
	suite.Regexp("-m: expected a key sequence after '.'", err)
}

func (suite *MetaPrimaryTestSuite) TestMetaPrimaryValidInput() {
	p, tokens, err := metaPrimary.parse([]string{"-m", "-empty", "-size"})
	if suite.NoError(err) {
		suite.Equal([]string{"-size"}, tokens)
		e := types.Entry{}
		suite.True(p(e))
	}
}

func TestMetaPrimary(t *testing.T) {
	suite.Run(t, new(MetaPrimaryTestSuite))
}
