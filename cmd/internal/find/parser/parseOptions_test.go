package parser

import (
	"flag"
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

// RTC => RunTestCase. Saves some typing
func (s *ParseOptionsTestSuite) RTC(input string, expectedOptions types.Options, expectedArgs string) {
	s.runTestCase(parseOptionsTestCase{
		input:           input,
		expectedOptions: expectedOptions,
		expectedArgs:    expectedArgs,
	})
}

// RETC => runErrorTestCase. Saves some typing
func (s *ParseOptionsTestSuite) RETC(input string, errRegex string) {
	s.runTestCase(parseOptionsTestCase{
		input:    input,
		errRegex: regexp.MustCompile(errRegex),
	})
}

func (s *ParseOptionsTestSuite) runTestCase(c parseOptionsTestCase) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input '%v'\n", input)
			panic(r)
		}
	}()
	args := []string{}
	input = c.input
	if input != "" {
		args = strings.Split(input, " ")
	}
	o, args, err := parseOptions(args)
	if c.errRegex != nil {
		s.Regexp(c.errRegex, err.Error(), "Input was '%v'", input)
	} else {
		if s.NoError(err) {
			s.Equal(c.expectedOptions, o)

			expectedArgs := []string{}
			if c.expectedArgs != "" {
				expectedArgs = strings.Split(c.expectedArgs, " ")
			}
			s.Equal(expectedArgs, args)
		}
	}
}

func (s *ParseOptionsTestSuite) TestParseOptionsNoArgs() {
	s.RTC("", types.NewOptions(), "")
}

func (s *ParseOptionsTestSuite) TestParseOptionsNoOptions() {
	o := types.NewOptions()
	s.RTC("--", o, "--")
	s.RTC("-true", o, "-true")
	s.RTC("-a", o, "-a")
	s.RTC("(", o, "(")
	s.RTC("foo bar baz", o, "foo bar baz")
}

func (s *ParseOptionsTestSuite) TestParseOptionInvalidOption() {
	s.RETC("-unknown", "flag.*unknown")
}

func (s *ParseOptionsTestSuite) TestParseOptionHelpFlag() {
	for _, helpFlag := range []string{"-h", "-help"} {
		o, _, err := parseOptions([]string{helpFlag})
		if s.Equal(flag.ErrHelp, err) {
			s.True(o.Help.Requested)
			s.False(o.Help.HasValue)
		}

		o, _, err = parseOptions([]string{helpFlag, ""})
		if s.Equal(flag.ErrHelp, err) {
			s.True(o.Help.Requested)
			s.False(o.Help.HasValue)
		}

		o, _, err = parseOptions([]string{helpFlag, "-maxdepth"})
		if s.Equal(flag.ErrHelp, err) {
			s.True(o.Help.Requested)
			s.False(o.Help.HasValue)
		}

		o, _, err = parseOptions([]string{helpFlag, "foo"})
		if s.Equal(flag.ErrHelp, err) {
			s.True(o.Help.Requested)
			s.True(o.Help.HasValue)
			s.False(o.Help.Syntax)
			s.Equal("foo", o.Help.Primary)
		}

		o, _, err = parseOptions([]string{helpFlag, "syntax"})
		if s.Equal(flag.ErrHelp, err) {
			s.True(o.Help.Requested)
			s.True(o.Help.HasValue)
			s.True(o.Help.Syntax)
		}
	}
}

func (s *ParseOptionsTestSuite) TestParseOptionsValidOptions() {
	o := types.NewOptions()
	o.Mindepth = 5
	o.MarkAsSet(types.MindepthFlag)
	s.RTC("-mindepth 5", o, "")
	s.RTC("-mindepth 5 --", o, "--")
	s.RTC("-mindepth 5 -true", o, "-true")
	s.RTC("-mindepth 5 -a", o, "-a")
	s.RTC("-mindepth 5 foo bar baz", o, "foo bar baz")
}

func (s *ParseOptionsTestSuite) TestParseOptionsNegativeMaxdepth() {
	o := types.NewOptions()
	o.MarkAsSet(types.MaxdepthFlag)
	s.RTC("-maxdepth -1", o, "")
}

func TestParseOptions(t *testing.T) {
	suite.Run(t, new(ParseOptionsTestSuite))
}
