package cmdfind

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

// The primaries are tested separately in their individual *Primary.go files, so they
// will not be tested here. Instead, the tests here serve as "integration tests" for
// the exported ParsePredicate function. They're meant to test parser errors, each of
// the operators, and whether operator precedence is enforced.
type ParserTestSuite struct {
	suite.Suite
}

type parserTestCase struct {
	input    string
	expected bool
	err      string
}

// nPTC => newParserTestCase. This helper saves some typing
func nPTC(input string, v interface{}) parserTestCase {
	ptc := parserTestCase{input: input}
	if bv, ok := v.(bool); ok {
		ptc.expected = bv
	} else {
		ptc.err = v.(string)
	}
	return ptc
}

func (suite *ParserTestSuite) runTestCases(testCases ...parserTestCase) {
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
		p, err := ParsePredicate(tks)
		if c.err != "" {
			suite.Equal(c.err, err.Error(), "Input was '%v'", input)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, p(newEntry()), "Input was '%v'", input)
			}
		}
	}
}

func (suite *ParserTestSuite) TestParsePredicateEmptyTokens() {
	suite.runTestCases(nPTC("", true))
}

func (suite *ParserTestSuite) TestParsePredicateNotOpParseErrors() {
	suite.runTestCases(
		// Test error when "-not" is supplied without an expression
		nPTC("-not", "-not: no following expression"),
		// Test error when "-not" is supplied w/ an atom that errors
		nPTC("-not -name", "-name: requires additional arguments"),
		// Test error when "-not" is followed by a binary operator
		nPTC("-not -a", "-not: no following expression"),
	)
}

func (suite *ParserTestSuite) TestParsePredicateNotOpEval() {
	suite.runTestCases(
		nPTC("-not -true", false),
		nPTC("-not -not -true", true),
		nPTC("-not -not -not -true", false),
	)
}

func (suite *ParserTestSuite) TestParsePredicateBinOpParseErrors() {
	suite.runTestCases(
		// Tests for -and
		nPTC("-a", "-a: no expression before -a"),
		nPTC("-true -a", "-a: no expression after -a"),
		nPTC("-true -a -a", "-a: no expression after -a"),
		// Tests for -or
		nPTC("-o", "-o: no expression before -o"),
		nPTC("-true -o", "-o: no expression after -o"),
		nPTC("-true -o -o", "-o: no expression after -o"),
	)
}

func (suite *ParserTestSuite) TestParsePredicateAndOpEval() {
	suite.runTestCases(
		nPTC("-true -a -false", false),
		nPTC("-true -false", false),
		nPTC("-true -true", true),
	)
}

func (suite *ParserTestSuite) TestParsePredicateOrOpEval() {
	suite.runTestCases(
		nPTC("-true -o -false", true),
		nPTC("-false -o -true", true),
	)
}

func (suite *ParserTestSuite) TestParsePredicateBinOpPrecedence() {
	suite.runTestCases(
		// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
		// Without precedence, this would be parsed as ((-true -o -true) -a false) which
		// evaluates to false.
		nPTC("-true -o -true -a -false", true),
	)
}

func (suite *ParserTestSuite) TestParsePredicateUnknownPrimaryOrOperatorError() {
	suite.runTestCases(nPTC("-foo", "-foo: unknown primary or operator"))
}

func (suite *ParserTestSuite) TestParsePredicateParensErrors() {
	suite.runTestCases(
		// Test the simple error cases
		nPTC(")", "): no beginning '('"),
		nPTC("(", "(: missing closing ')'"),
		nPTC("( )", "(): empty inner expression"),
		// Test some more complicated error cases
		nPTC("( -true ) )", "): no beginning '('"),
		nPTC("( -true ) ( ) -true", "(): empty inner expression"),
		nPTC("( -true ( -false )", "(: missing closing ')'"),
		nPTC("( ( ( -true ) ) ) )", "): no beginning '('"),
		nPTC("( -a )", "-a: no expression before -a"),
		nPTC("( ( ( -true ) -a", "(: missing closing ')'"),
		nPTC("( ( ( -true ) -a ) )", "-a: no expression after -a"),
	)
}

func (suite *ParserTestSuite) TestParsePredicateParensEval() {
	suite.runTestCases(
		// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
		// which would evaluate to true.
		nPTC("( -true -o -true ) -a -false", false),
		nPTC("-not ( -true -o -false )", false),
		nPTC("( -true ) -a ( -false )", false),
		nPTC("( -true ( -false ) -o ( ( -false -true ) ) )", false),
		nPTC("( ( ( -true ) ) )", true),
		nPTC("( ( -true ) -a -false )", false),
	)
}

func (suite *ParserTestSuite) TestParsePredicateComplexErrors() {
	suite.runTestCases(
		nPTC("( -true ) -a )", "): no beginning '('"),
	)
}

func (suite *ParserTestSuite) TestParsePredicateComplexEval() {
	suite.runTestCases(
		nPTC("( -true -o -true ) -false", false),
		// Should be parsed as (-true -a -false) -o -true which evaluates to true.
		nPTC("-true -false -o -true", true),
		nPTC("-false -o -true -false", false),
		nPTC("( -true -true ) -o -false", true),
	)
}

func TestParser(t *testing.T) {
	suite.Run(t, new(ParserTestSuite))
}
