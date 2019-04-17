package parser

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParsePathTestSuite struct {
	suite.Suite
}

func (suite *ParsePathTestSuite) TestParsePath() {
	// No args
	path, args := parsePath([]string{})
	suite.Equal(defaultPath, path)
	suite.Equal([]string{}, args)

	// Empty path
	path, args = parsePath([]string{""})
	suite.Equal(defaultPath, path)
	suite.Equal([]string{}, args)

	// When args contains only `wash find`'s expression
	path, args = parsePath([]string{"-true"})
	suite.Equal(defaultPath, path)
	suite.Equal([]string{"-true"}, args)

	// When args does contain a path
	path, args = parsePath([]string{"foo", "-true"})
	suite.Equal("foo", path)
	suite.Equal([]string{"-true"}, args)
}

func TestParsePath(t *testing.T) {
	suite.Run(t, new(ParsePathTestSuite))
}
