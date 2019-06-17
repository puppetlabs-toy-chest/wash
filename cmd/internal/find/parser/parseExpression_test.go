package parser

import (
	"regexp"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

// The primaries are tested separately in their individual primary/*.go files, so they
// will not be tested here. Instead, the tests here serve as "integration tests" for
// the parseExpression function. They're meant to test parser errors, each of
// the operators, and whether operator precedence is enforced.
type ParseExpressionTestSuite struct {
	parsertest.Suite
}

func (s *ParseExpressionTestSuite) RETC(input string, errMsg string) {
	s.Suite.RETC(input, regexp.QuoteMeta(errMsg), false)
}

func (s *ParseExpressionTestSuite) RTC(input string, expected bool) {
	if expected {
		s.Suite.RTC(input, "", types.Entry{})
	} else {
		s.Suite.RNTC(input, "", types.Entry{})
	}
}

func (s *ParseExpressionTestSuite) TestParseExpressionEmptyTokens() {
	s.RTC("", true)
}

func (s *ParseExpressionTestSuite) TestParseExpressionNotOpParseErrors() {
	// Test error when "-not" is supplied without an expression
	s.RETC("-not", "-not: no following expression")
	// Test error when "-not" is mixed with parentheses
	s.RETC("-not )", "): no beginning '('")
	s.RETC("( -not )", "-not: no following expression")
	// Test error when "-not" is supplied w/ an atom that errors
	s.RETC("-not -name", "-name: requires additional arguments")
	// Test error when "-not" is followed by a binary operator
	s.RETC("-not -a", "-not: no following expression")
}

func (s *ParseExpressionTestSuite) TestParseExpressionNotOpEval() {
	s.RTC("-not -true", false)
	s.RTC("-not -not -true", true)
	s.RTC("-not -not -not -true", false)
}

func (s *ParseExpressionTestSuite) TestParseExpressionBinOpParseErrors() {
	// Tests for -and
	s.RETC("-a", "-a: no expression before -a")
	s.RETC("-true -a", "-a: no expression after -a")
	s.RETC("-true -a -a", "-a: no expression after -a")
	// Tests for -or
	s.RETC("-o", "-o: no expression before -o")
	s.RETC("-true -o", "-o: no expression after -o")
	s.RETC("-true -o -o", "-o: no expression after -o")
}

func (s *ParseExpressionTestSuite) TestParseExpressionAndOpEval() {
	s.RTC("-false -a -false", false)
	s.RTC("-false -false", false)
	s.RTC("-false -a -true", false)
	s.RTC("-false -true", false)
	s.RTC("-true -a -false", false)
	s.RTC("-true -false", false)
	s.RTC("-true -a -true", true)
	s.RTC("-true -true", true)
}

func (s *ParseExpressionTestSuite) TestParseExpressionOrOpEval() {
	s.RTC("-false -o -false", false)
	s.RTC("-false -o -true", true)
	s.RTC("-true -o -false", true)
	s.RTC("-true -o -true", true)
}

func (s *ParseExpressionTestSuite) TestParseExpressionBinOpPrecedence() {
	// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
	// Without precedence, this would be parsed as ((-true -o -true) -a false) which
	// evaluates to false.
	s.RTC("-true -o -true -a -false", true)
}

func (s *ParseExpressionTestSuite) TestParseExpressionUnknownPrimaryOrOperatorError() {
	s.RETC("-foo", "-foo: unknown primary or operator")
}

func (s *ParseExpressionTestSuite) TestParseExpressionParensErrors() {
	// Test the simple error cases
	s.RETC(")", "): no beginning '('")
	s.RETC("(", "(: missing closing ')'")
	s.RETC("( )", "(): empty inner expression")
	// Test some more complicated error cases
	s.RETC("( -true ) )", "): no beginning '('")
	s.RETC("( -true ) ( ) -true", "(): empty inner expression")
	s.RETC("( -true ( -false )", "(: missing closing ')'")
	s.RETC("( ( ( -true ) ) ) )", "): no beginning '('")
	s.RETC("( -a )", "-a: no expression before -a")
	s.RETC("( ( ( -true ) -a", "-a: no expression after -a")
	s.RETC("( ( ( -true ) -a ) )", "-a: no expression after -a")
}

func (s *ParseExpressionTestSuite) TestParseExpressionParensEval() {
	// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
	// which would evaluate to true.
	s.RTC("( -true -o -true ) -a -false", false)
	s.RTC("-not ( -true -o -false )", false)
	s.RTC("( -true ) -a ( -false )", false)
	s.RTC("( -true ( -false ) -o ( ( -false -true ) ) )", false)
	s.RTC("( ( ( -true ) ) )", true)
	s.RTC("( ( -true ) -a -false )", false)
}

func (s *ParseExpressionTestSuite) TestParseExpressionComplexErrors() {
	s.RETC("( -true ) -a )", "): no beginning '('")
	s.RETC("-true -a -foo", "-foo: unknown primary or operator")
	// Make sure meta primary expressions are parsed independently of
	// the top-level `wash find` expression
	s.RETC("-m .key (", "-m: (: missing closing ')'")
}

func (s *ParseExpressionTestSuite) TestParseExpressionComplexEval() {
	// Set-up the entry for the meta primary integration test
	m := make(map[string]interface{})
	m["key"] = "foo"
	entry := types.Entry{}
	entry.Metadata = m

	s.RTC("( -true -o -true ) -false", false)
	// Should be parsed as (-true -a -false) -o -true which evaluates to true.
	s.RTC("-true -false -o -true", true)
	s.RTC("-false -o -true -false", false)
	s.RTC("( -true -true ) -o -false", true)
	// Test meta primary integration. Use s.Suite.RTC/s.Suite.NPNTC because
	// we're providing our own entry
	s.Suite.RTC("-m .key foo -o -m .key bar", "", entry)
	s.Suite.RNTC("-m .key foo -a -m .key bar", "", entry)
}

func TestParseExpression(t *testing.T) {
	s := new(ParseExpressionTestSuite)
	s.Parser = types.EntryPredicateParser(func(tokens []string) (*types.EntryPredicate, []string, error) {
		p, err := parseExpression(tokens)
		return p, []string{}, err
	})
	suite.Run(t, s)
}
