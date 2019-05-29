package parser

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ParsePathsTestSuite struct {
	suite.Suite
}

func (suite *ParsePathsTestSuite) TestParsePaths() {
	// tc => testCase. Saves some typing.
	type tc struct {
		input    []string
		remInput []string
		paths    []string
	}
	cases := []tc{
		// No args
		tc{[]string{}, []string{}, []string{defaultPath}},
		// Empty args
		tc{[]string{"", ""}, []string{}, []string{defaultPath}},
		// When args contains only the options
		tc{[]string{"-maxdepth"}, []string{"-maxdepth"}, []string{defaultPath}},
		// When args contains only `wash find`'s expression
		tc{[]string{"-true"}, []string{"-true"}, []string{defaultPath}},
		tc{[]string{"("}, []string{"("}, []string{defaultPath}},
		// When args contains both options and `wash find`'s expression
		tc{[]string{"-maxdepth", "("}, []string{"-maxdepth", "("}, []string{defaultPath}},
		// When args contains a path
		tc{[]string{"foo", "-maxdepth", "-true"}, []string{"-maxdepth", "-true"}, []string{"foo"}},
		// When args contains multiple paths
		tc{[]string{"foo", "bar", "baz", "-maxdepth", "-true"}, []string{"-maxdepth", "-true"}, []string{"foo", "bar", "baz"}},
	}
	for _, c := range cases {
		inputStr := fmt.Sprintf("Input was: %v", c.input)
		paths, args := parsePaths(c.input)
		suite.Equal(c.paths, paths, inputStr)
		suite.Equal(c.remInput, args, inputStr)		
	}
}

func TestParsePaths(t *testing.T) {
	suite.Run(t, new(ParsePathsTestSuite))
}
