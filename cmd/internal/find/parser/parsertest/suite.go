package parsertest

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

// Suite represents a type that tests predicate parsers
type Suite struct {
	suite.Suite
	Parser predicate.Parser
}

// Case represents a parser test case
type Case struct {
	Input           string
	RemInput        string
	SatisfyingValue interface{}
	ErrRegex        *regexp.Regexp
	IsMatchError    bool
}

// NPTC => NewCase. Saves some typing
func (suite *Suite) NPTC(input string, remInput string, trueValue interface{}) Case {
	return Case{
		Input:           input,
		RemInput:        remInput,
		SatisfyingValue: trueValue,
	}
}

// NPNTC => NewParserNegativeTestCase. Saves some typing
func (suite *Suite) NPNTC(input string, remInput string, falseValue interface{}) Case {
	return Case{
		Input:           input,
		RemInput:        remInput,
		SatisfyingValue: falseV{falseValue},
	}
}

// NPETC => NewParserErrorTestCase
func (suite *Suite) NPETC(input string, errRegex string, isMatchError bool) Case {
	return Case{
		Input:        input,
		ErrRegex:     regexp.MustCompile(errRegex),
		IsMatchError: isMatchError,
	}
}

// RunTestCases runs the given test cases.
func (suite *Suite) RunTestCases(cases ...Case) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input: %v\n", input)
			panic(r)
		}
	}()
	for _, c := range cases {
		input = c.Input
		p, tokens, err := suite.Parser.Parse(suite.ToTks(input))
		if c.ErrRegex != nil {
			if c.IsMatchError {
				suite.True(errz.IsMatchError(err), "Input %v: expected an errz.MatchError", input)
			} else {
				suite.False(errz.IsMatchError(err), "Input %v: received an unexpected errz.MatchError", input)
			}
			suite.Regexp(c.ErrRegex, err, "Input: %v", input)
		} else {
			if suite.NoError(err, "Input: %v", input) {
				suite.Equal(suite.ToTks(c.RemInput), tokens, "Input: %v", input)
				falseV, ok := c.SatisfyingValue.(falseV)
				if ok {
					suite.False(p.IsSatisfiedBy(falseV.v), "Input: %v, Value: %t", input, falseV.v)
				} else {
					suite.True(p.IsSatisfiedBy(c.SatisfyingValue), "Input: %v, Value: %t", input, c.SatisfyingValue)
				}
			}
		}
	}
}

// ToTks => ToTokens. Saves some typing
func (suite *Suite) ToTks(s string) []string {
	var tokens = []string{}
	if s != "" {
		tokens = strings.Split(s, " ")
	}
	return tokens
}

// SetupTest sets the StartTime
func (suite *Suite) SetupTest() {
	params.StartTime = time.Now()
}

// TeardownTest resets the StartTime
func (suite *Suite) TeardownTest() {
	params.StartTime = time.Time{}
}


// falseV's a wrapper type that's used to distingush between "positive" and "negative"
// satisfying values. We need it b/c "nil" could be a satisfying value.
type falseV struct {
	v interface{}
}
