package parser

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type ParseOptionsTestSuite struct {
	suite.Suite
}

type parseOptionsTestCase struct {
	input           string
	expectedOptions types.Options
	expectedArgs    string
	errRegex        *regexp.Regexp
}

// nPOTC => newParseOptionsTestCase. Saves some typing
func nPOTC(input string, expectedOptions types.Options, expectedArgs string) parseOptionsTestCase {
	return parseOptionsTestCase{
		input:           input,
		expectedOptions: expectedOptions,
		expectedArgs:    expectedArgs,
	}
}

// nPOETC => newParseOptionsErrorTestCase. Saves some typing
func nPOETC(input string, errRegex string) parseOptionsTestCase {
	return parseOptionsTestCase{
		input:    input,
		errRegex: regexp.MustCompile(errRegex),
	}
}

func (suite *ParseOptionsTestSuite) runTestCases(testCases ...parseOptionsTestCase) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input '%v'\n", input)
			panic(r)
		}
	}()
	for _, c := range testCases {
		args := []string{}
		input = c.input
		if input != "" {
			args = strings.Split(input, " ")
		}
		o, args, err := parseOptions(args)
		if c.errRegex != nil {
			suite.Regexp(c.errRegex, err.Error(), "Input was '%v'", input)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expectedOptions, o)

				expectedArgs := []string{}
				if c.expectedArgs != "" {
					expectedArgs = strings.Split(c.expectedArgs, " ")
				}
				suite.Equal(expectedArgs, args)
			}
		}
	}
}

func (suite *ParseOptionsTestSuite) TestParseOptionsNoArgs() {
	suite.runTestCases(nPOTC("", types.NewOptions(), ""))
}

func (suite *ParseOptionsTestSuite) TestParseOptionsNoOptions() {
	o := types.NewOptions()
	suite.runTestCases(
		nPOTC("--", o, "--"),
		nPOTC("-true", o, "-true"),
		nPOTC("-a", o, "-a"),
		nPOTC("(", o, "("),
		nPOTC("foo bar baz", o, "foo bar baz"),
	)
}

func (suite *ParseOptionsTestSuite) TestParseOptionInvalidOption() {
	suite.runTestCases(nPOETC("-unknown", "flag.*unknown"))
}

func (suite *ParseOptionsTestSuite) TestParseOptionsValidOptions() {
	o := types.NewOptions()
	o.Mindepth = 5
	o.MarkAsSet(types.MindepthFlag)
	suite.runTestCases(
		nPOTC("-mindepth 5", o, ""),
		nPOTC("-mindepth 5 --", o, "--"),
		nPOTC("-mindepth 5 -true", o, "-true"),
		nPOTC("-mindepth 5 -a", o, "-a"),
		nPOTC("-mindepth 5 foo bar baz", o, "foo bar baz"),
	)
}

func TestParseOptions(t *testing.T) {
	suite.Run(t, new(ParseOptionsTestSuite))
}
