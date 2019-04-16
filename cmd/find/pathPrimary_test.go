package cmdfind

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type PathPrimaryTestSuite struct {
	suite.Suite
}

func (suite *PathPrimaryTestSuite) TestPathPrimaryErrors() {
	_, _, err := pathPrimary.parse([]string{"-path"})
	suite.Regexp("-path: requires additional arguments", err)

	_, _, err = pathPrimary.parse([]string{"-path", "[a"})
	suite.Regexp("-path: invalid glob: unexpected end of input", err)
}

func (suite *PathPrimaryTestSuite) TestPathPrimaryValidInput() {
	e1 := newEntry()
	e1.NormalizedPath = "a"
	e2 := newEntry()
	e2.NormalizedPath = "b"
	p, tokens, err := pathPrimary.parse([]string{"-path", "a"})
	if suite.NoError(err) {
		suite.Equal([]string{}, tokens)
		suite.Equal(true, p(e1))
		suite.Equal(false, p(e2))
	}
}

func TestPathPrimary(t *testing.T) {
	suite.Run(t, new(PathPrimaryTestSuite))
}
