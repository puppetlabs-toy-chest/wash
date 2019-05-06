package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

// The primaries are tested separately in their individual primary/*.go files, so they
// will not be tested here. Instead, the tests here serve as "integration tests" for
// the parseExpression function. They're meant to test parser errors, each of
// the operators, and whether operator precedence is enforced.
type ParseExpressionTestSuite struct {
	suite.Suite
}

type parseExpressionTestCase struct {
	input    string
	expected bool
	err      string
}

// nPETC => newParseExpressionTestCase. This helper saves some typing
func nPETC(input string, v interface{}) parseExpressionTestCase {
	ptc := parseExpressionTestCase{input: input}
	if bv, ok := v.(bool); ok {
		ptc.expected = bv
	} else {
		ptc.err = v.(string)
	}
	return ptc
}

func (suite *ParseExpressionTestSuite) runTestCases(testCases ...parseExpressionTestCase) {
	var input string
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panicked on input '%v'\n", input)
			panic(r)
		}
	}()
	for _, c := range testCases {
		tks := []string{}
		input = c.input
		if input != "" {
			tks = strings.Split(input, " ")
		}
		p, err := parseExpression(tks)
		if c.err != "" {
			suite.Equal(c.err, err.Error(), "Input was '%v'", input)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, p(types.Entry{}), "Input was '%v'", input)
			}
		}
	}
}

func (suite *ParseExpressionTestSuite) TestParseExpressionEmptyTokens() {
	suite.runTestCases(nPETC("", true))
}

func (suite *ParseExpressionTestSuite) TestParseExpressionNotOpParseErrors() {
	suite.runTestCases(
		// Test error when "-not" is supplied without an expression
		nPETC("-not", "-not: no following expression"),
		// Test error when "-not" is mixed with parentheses
		nPETC("-not )", "): no beginning '('"),
		nPETC("( -not )", "-not: no following expression"),
		// Test error when "-not" is supplied w/ an atom that errors
		nPETC("-not -name", "-name: requires additional arguments"),
		// Test error when "-not" is followed by a binary operator
		nPETC("-not -a", "-not: no following expression"),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionNotOpEval() {
	suite.runTestCases(
		nPETC("-not -true", false),
		nPETC("-not -not -true", true),
		nPETC("-not -not -not -true", false),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionBinOpParseErrors() {
	suite.runTestCases(
		// Tests for -and
		nPETC("-a", "-a: no expression before -a"),
		nPETC("-true -a", "-a: no expression after -a"),
		nPETC("-true -a -a", "-a: no expression after -a"),
		// Tests for -or
		nPETC("-o", "-o: no expression before -o"),
		nPETC("-true -o", "-o: no expression after -o"),
		nPETC("-true -o -o", "-o: no expression after -o"),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionAndOpEval() {
	suite.runTestCases(
		nPETC("-true -a -false", false),
		nPETC("-true -false", false),
		nPETC("-true -true", true),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionOrOpEval() {
	suite.runTestCases(
		nPETC("-true -o -false", true),
		nPETC("-false -o -true", true),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionBinOpPrecedence() {
	suite.runTestCases(
		// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
		// Without precedence, this would be parsed as ((-true -o -true) -a false) which
		// evaluates to false.
		nPETC("-true -o -true -a -false", true),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionUnknownPrimaryOrOperatorError() {
	suite.runTestCases(nPETC("-foo", "-foo: unknown primary or operator"))
}

func (suite *ParseExpressionTestSuite) TestParseExpressionParensErrors() {
	suite.runTestCases(
		// Test the simple error cases
		nPETC(")", "): no beginning '('"),
		nPETC("(", "(: missing closing ')'"),
		nPETC("( )", "(): empty inner expression"),
		// Test some more complicated error cases
		nPETC("( -true ) )", "): no beginning '('"),
		nPETC("( -true ) ( ) -true", "(): empty inner expression"),
		nPETC("( -true ( -false )", "(: missing closing ')'"),
		nPETC("( ( ( -true ) ) ) )", "): no beginning '('"),
		nPETC("( -a )", "-a: no expression before -a"),
		nPETC("( ( ( -true ) -a", "(: missing closing ')'"),
		nPETC("( ( ( -true ) -a ) )", "-a: no expression after -a"),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionParensEval() {
	suite.runTestCases(
		// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
		// which would evaluate to true.
		nPETC("( -true -o -true ) -a -false", false),
		nPETC("-not ( -true -o -false )", false),
		nPETC("( -true ) -a ( -false )", false),
		nPETC("( -true ( -false ) -o ( ( -false -true ) ) )", false),
		nPETC("( ( ( -true ) ) )", true),
		nPETC("( ( -true ) -a -false )", false),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionComplexErrors() {
	suite.runTestCases(
		nPETC("( -true ) -a )", "): no beginning '('"),
		nPETC("-true -a -foo", "-foo: unknown primary or operator"),
	)
}

func (suite *ParseExpressionTestSuite) TestParseExpressionComplexEval() {
	suite.runTestCases(
		nPETC("( -true -o -true ) -false", false),
		// Should be parsed as (-true -a -false) -o -true which evaluates to true.
		nPETC("-true -false -o -true", true),
		nPETC("-false -o -true -false", false),
		nPETC("( -true -true ) -o -false", true),
	)
}

func TestParseExpression(t *testing.T) {
	suite.Run(t, new(ParseExpressionTestSuite))
}
