package meta

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
	"github.com/stretchr/testify/suite"
)

// This file stores some common test code that's shared by all the
// *Predicate.go files.

type ParserTestSuite struct {
	suite.Suite
	parser predicateParser
}

type parserTestCase struct {
	input           string
	rem             string
	satisfyingValue interface{}
	errRegex        *regexp.Regexp
	isMatchError    bool
}

// nPTC => newParserTestCase. Saves some typing
func nPTC(input string, rem string, satisfyingValue interface{}) parserTestCase {
	return parserTestCase{
		input:           input,
		rem:             rem,
		satisfyingValue: satisfyingValue,
	}
}

// nPETC => newParserErrorTestCase
func nPETC(input string, errRegex string, isMatchError bool) parserTestCase {
	return parserTestCase{
		input:        input,
		errRegex:     regexp.MustCompile(errRegex),
		isMatchError: isMatchError,
	}
}

func (suite *ParserTestSuite) runTestCases(cases ...parserTestCase) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input: %v\n", input)
			panic(r)
		}
	}()
	for _, c := range cases {
		input = c.input
		p, tokens, err := suite.parser(toTks(input))
		if c.errRegex != nil {
			if c.isMatchError {
				suite.True(errz.IsMatchError(err), "Input %v: expected an errz.MatchError", input)
			} else {
				suite.False(errz.IsMatchError(err), "Input %v: received an unexpected errz.MatchError", input)
			}
			suite.Regexp(c.errRegex, err, "Input: %v", input)
		} else {
			if suite.NoError(err, "Input: %v", input) {
				suite.Equal(toTks(c.rem), tokens, "Input: %v", input)
				suite.True(p(c.satisfyingValue), "Input: %v, Value: %t", input, c.satisfyingValue)
			}
		}
	}
}

// toTks => toTokens. Saves some typing
func toTks(s string) []string {
	var tokens = []string{}
	if s != "" {
		tokens = strings.Split(s, " ")
	}
	return tokens
}
