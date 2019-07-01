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

// Suite represents a type that tests `wash find` predicate parsers
type Suite struct {
	suite.Suite
	Parser        predicate.Parser
	SchemaPParser predicate.Parser
}

// Case represents a parser test case
type Case struct {
	Input           string
	RemInput        string
	SatisfyingValue interface{}
	ErrRegex        *regexp.Regexp
	IsMatchError    bool
	parser          predicate.Parser
}

// RTC => RunTestCase. Saves some typing
func (suite *Suite) RTC(input string, remInput string, trueValue interface{}, falseValue ...interface{}) {
	suite.rTC(suite.Parser, input, remInput, trueValue)
	if len(falseValue) > 0 {
		suite.RNTC(input, remInput, falseValue[0])
	}
}

// RNTC => RunNegativeTestCase. Saves some typing
func (suite *Suite) RNTC(input string, remInput string, falseValue interface{}) {
	suite.rTC(suite.Parser, input, remInput, falseV{falseValue})
}

// RSTC => RunSchemaTestCase. Saves some typing.
func (suite *Suite) RSTC(input string, remInput string, trueValue interface{}, falseValue ...interface{}) {
	suite.rTC(suite.SchemaPParser, input, remInput, trueValue)
	if len(falseValue) > 0 {
		suite.rTC(suite.SchemaPParser, input, remInput, falseValue[0])
	}
}

// RNSTC => RunNegativeSchemaTestCase. Saves some typing
func (suite *Suite) RNSTC(input string, remInput string, falseValue interface{}) {
	suite.rTC(suite.SchemaPParser, input, remInput, falseV{falseValue})
}

// RETC => RunErrorTestCase
func (suite *Suite) RETC(input string, errRegex string, isMatchError bool) {
	suite.runTestCase(Case{
		Input:        input,
		ErrRegex:     regexp.MustCompile(errRegex),
		IsMatchError: isMatchError,
		parser:       suite.Parser,
	})
}

func (suite *Suite) rTC(parser predicate.Parser, input string, remInput string, trueValue interface{}) {
	suite.runTestCase(Case{
		Input:           input,
		RemInput:        remInput,
		SatisfyingValue: trueValue,
		parser:          parser,
	})
}

// runTestCase runs the given test case
func (suite *Suite) runTestCase(c Case) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input: %v\n", input)
			panic(r)
		}
	}()
	input = c.Input
	p, tokens, err := c.parser.Parse(suite.ToTks(input))
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

// ToTks => ToTokens. Saves some typing
func (suite *Suite) ToTks(s string) []string {
	var tokens = []string{}
	if s != "" {
		tokens = strings.Split(s, " ")
	}
	return tokens
}

// SetupTest sets the ReferenceTime
func (suite *Suite) SetupTest() {
	params.ReferenceTime = time.Now()
}

// TeardownTest resets the ReferenceTime
func (suite *Suite) TeardownTest() {
	params.ReferenceTime = time.Time{}
}

// falseV's a wrapper type that's used to distingush between "positive" and "negative"
// satisfying values. We need it b/c "nil" could be a satisfying value.
type falseV struct {
	v interface{}
}
