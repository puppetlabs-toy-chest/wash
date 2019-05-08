package parser

import (
	"regexp"
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/params"
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

func (s *ParseExpressionTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *ParseExpressionTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (s *ParseExpressionTestSuite) NPETC(input string, errMsg string) parsertest.Case {
	return s.Suite.NPETC(input, regexp.QuoteMeta(errMsg), false)
}

func (s *ParseExpressionTestSuite) NPTC(input string, expected bool) parsertest.Case {
	if expected {
		return s.Suite.NPTC(input, "", types.Entry{})
	}
	return s.Suite.NPNTC(input, "", types.Entry{})
}

func (s *ParseExpressionTestSuite) TestParseExpressionEmptyTokens() {
	s.RunTestCases(s.NPTC("", true))
}

func (s *ParseExpressionTestSuite) TestParseExpressionNotOpParseErrors() {
	s.RunTestCases(
		// Test error when "-not" is supplied without an expression
		s.NPETC("-not", "-not: no following expression"),
		// Test error when "-not" is mixed with parentheses
		s.NPETC("-not )", "): no beginning '('"),
		s.NPETC("( -not )", "-not: no following expression"),
		// Test error when "-not" is supplied w/ an atom that errors
		s.NPETC("-not -name", "-name: requires additional arguments"),
		// Test error when "-not" is followed by a binary operator
		s.NPETC("-not -a", "-not: no following expression"),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionNotOpEval() {
	s.RunTestCases(
		s.NPTC("-not -true", false),
		s.NPTC("-not -not -true", true),
		s.NPTC("-not -not -not -true", false),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionBinOpParseErrors() {
	s.RunTestCases(
		// Tests for -and
		s.NPETC("-a", "-a: no expression before -a"),
		s.NPETC("-true -a", "-a: no expression after -a"),
		s.NPETC("-true -a -a", "-a: no expression after -a"),
		// Tests for -or
		s.NPETC("-o", "-o: no expression before -o"),
		s.NPETC("-true -o", "-o: no expression after -o"),
		s.NPETC("-true -o -o", "-o: no expression after -o"),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionAndOpEval() {
	s.RunTestCases(
		s.NPTC("-false -a -false", false),
		s.NPTC("-false -false", false),
		s.NPTC("-false -a -true", false),
		s.NPTC("-false -true", false),
		s.NPTC("-true -a -false", false),
		s.NPTC("-true -false", false),
		s.NPTC("-true -a -true", true),
		s.NPTC("-true -true", true),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionOrOpEval() {
	s.RunTestCases(
		s.NPTC("-false -o -false", false),
		s.NPTC("-false -o -true", true),
		s.NPTC("-true -o -false", true),
		s.NPTC("-true -o -true", true),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionBinOpPrecedence() {
	s.RunTestCases(
		// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
		// Without precedence, this would be parsed as ((-true -o -true) -a false) which
		// evaluates to false.
		s.NPTC("-true -o -true -a -false", true),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionUnknownPrimaryOrOperatorError() {
	s.RunTestCases(s.NPETC("-foo", "-foo: unknown primary or operator"))
}

func (s *ParseExpressionTestSuite) TestParseExpressionParensErrors() {
	s.RunTestCases(
		// Test the simple error cases
		s.NPETC(")", "): no beginning '('"),
		s.NPETC("(", "(: missing closing ')'"),
		s.NPETC("( )", "(): empty inner expression"),
		// Test some more complicated error cases
		s.NPETC("( -true ) )", "): no beginning '('"),
		s.NPETC("( -true ) ( ) -true", "(): empty inner expression"),
		s.NPETC("( -true ( -false )", "(: missing closing ')'"),
		s.NPETC("( ( ( -true ) ) ) )", "): no beginning '('"),
		s.NPETC("( -a )", "-a: no expression before -a"),
		s.NPETC("( ( ( -true ) -a", "(: missing closing ')'"),
		s.NPETC("( ( ( -true ) -a ) )", "-a: no expression after -a"),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionParensEval() {
	s.RunTestCases(
		// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
		// which would evaluate to true.
		s.NPTC("( -true -o -true ) -a -false", false),
		s.NPTC("-not ( -true -o -false )", false),
		s.NPTC("( -true ) -a ( -false )", false),
		s.NPTC("( -true ( -false ) -o ( ( -false -true ) ) )", false),
		s.NPTC("( ( ( -true ) ) )", true),
		s.NPTC("( ( -true ) -a -false )", false),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionComplexErrors() {
	s.RunTestCases(
		s.NPETC("( -true ) -a )", "): no beginning '('"),
		s.NPETC("-true -a -foo", "-foo: unknown primary or operator"),
		// Make sure meta primary expressions are parsed independently of
		// the top-level `wash find` expression
		s.NPETC("-m .key (", "-m: (: missing closing ')'"),
	)
}

func (s *ParseExpressionTestSuite) TestParseExpressionComplexEval() {
	// Set-up the entry for the meta primary integration test
	m := make(map[string]interface{})
	m["key"] = "foo"
	entry := types.Entry{}
	entry.Attributes.SetMeta(m)

	s.RunTestCases(
		s.NPTC("( -true -o -true ) -false", false),
		// Should be parsed as (-true -a -false) -o -true which evaluates to true.
		s.NPTC("-true -false -o -true", true),
		s.NPTC("-false -o -true -false", false),
		s.NPTC("( -true -true ) -o -false", true),
		// Test meta primary integration. Use s.Suite.NPTC/s.Suite.NPNTC because
		// we're providing our own entry
		s.Suite.NPTC("-m .key foo -o -m .key bar", "", entry),
		s.Suite.NPNTC("-m .key foo -a -m .key bar", "", entry),
	)
}

func TestParseExpression(t *testing.T) {
	s := new(ParseExpressionTestSuite)
	s.Parser = types.EntryPredicateParser(func(tokens []string) (types.EntryPredicate, []string, error) {
		p, err := parseExpression(tokens)
		return p, []string{}, err
	})
	suite.Run(t, s)
}
