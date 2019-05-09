package meta

import (
	"regexp"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/stretchr/testify/suite"
)

// The predicates are tested separately in their *Predicate.go files, so they
// will not be tested here. Instead, the tests here serve as "integration tests" for
// the parsePredicateExpression function. They're meant to test parser errors, each of
// the operators, and whether operator precedence is enforced. 
//
// Note that inner expressions are tested in ObjectPredicate/ArrayPredicate since that
// is where they are used. They're also tested in the meta primary tests.

type PredicateExpressionTestSuite struct {
	parsertest.Suite
}

func (s *PredicateExpressionTestSuite) NPETC(input string, errRegex string) parsertest.Case {
	return s.Suite.NPETC(input, regexp.QuoteMeta(errRegex), false)
}

// NPOETC => NewParserOpEvalTestCase. input should only contain Boolean predicates (-true/-false).
// The Boolean predicates are meant to represent individual predicates that evaluate to -true/-false,
// respectively. Thus, NPOETC("-false -o -false", "", false) is read as "Test that if p1 returns false,
// and p2 returns false, then p1 -o p2 returns false". It is _not_ meant to be confused with
// NPTC("-false -o -false", "", false), which is read as "Test that the parsed predicate returns true if
// v is false OR is false". In other words, in NPTC, "-false" is a Boolean predicate while in NPOETC,
// "-false" represents _any_ predicate p (object/array/primitive/...) that returns false. NPOETC is useful
// to make binary op/parens op tests more expressive.
func (s *PredicateExpressionTestSuite) NPOETC(input string, remInput string, expectedValue bool) parsertest.Case {
	// -false/-true will still be parsed as Boolean predicates. Using "true" as the satisfying value in
	// both cases ensures that -false/-true represent an arbitrary predicate p that returns false/true,
	// respectively (since -false(true) == false, -true(true) == true).
	if expectedValue {
		return s.NPTC(input, remInput, true)
	}
	return s.NPNTC(input, remInput, true)
}

func (s *PredicateExpressionTestSuite) TestEmptyExpression() {
	s.RunTestCases(
		s.NPETC("", "expected a predicate expression"),
		s.NPETC("-primary", "unknown predicate -primary"),
	)
}

func (s *PredicateExpressionTestSuite) TestNotOpParseErrors() {
	s.RunTestCases(
		// Test error when "-not" is supplied without an expression
		s.NPETC("-not", "-not: no following expression"),
		// Test error when "-not" is mixed with parentheses
		s.NPETC("-not )", "): no beginning '('"),
		s.NPETC("( -not )", "-not: no following expression"),
		// Test error when "-not" is supplied w/ a malformed predicate
		s.NPETC("-not .", "expected a key sequence after '.'"),
		// Test error when "-not" is followed by a binary operator
		s.NPETC("-not -a", "-not: no following expression"),
	)
}

func (s *PredicateExpressionTestSuite) TestNotOpEval() {
	s.RunTestCases(
		s.NPTC("-not -true -primary", "-primary", false),
		s.NPTC("-not -not -true -primary", "-primary", true),
		s.NPTC("-not -not -not -true -primary", "-primary", false),
		// Ensure that "-not <predicate>" and "-not -not <predicate>" always
		// return false for mis-typed values. It is important to thoroughly
		// test this behavior because it can be very confusing for users to
		// debug if something goes wrong with the negation.
		//
		// Start with array predicates
		s.NPNTC("-not [?] -true -primary", "-primary", "foo"),
		s.NPNTC("-not -not [?] -true -primary", "-primary", "foo"),
		// empty predicate
		s.NPNTC("-not -empty -primary", "-primary", "foo"),
		s.NPNTC("-not -not -empty -primary", "-primary", "foo"),
		// numeric predicate
		s.NPNTC("-not +1 -primary", "-primary", "2"),
		s.NPNTC("-not -not -1 -primary", "-primary", "0"),
		// object predicate
		s.NPNTC("-not .key -true -primary", "-primary", "foo"),
		s.NPNTC("-not -not .key -true -primary", "-primary", "foo"),
		// boolean predicate
		s.NPNTC("-not -true -primary", "-primary", "false"),
		s.NPNTC("-not -not -true -primary", "-primary", "true"),
		// string predicate	
		s.NPNTC("-not f -primary", "-primary", 'g'),
		s.NPNTC("-not -not f -primary", "-primary", 'f'),
		// time predicate
		s.NPNTC("-not +1h -primary", "-primary", "not a time"),
		s.NPNTC("-not -not +1h -primary", "-primary", "not a time"),
	)
}

func (s *PredicateExpressionTestSuite) TestBinOpParseErrors() {
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

func (s *PredicateExpressionTestSuite) TestAndOpEval() {
	s.RunTestCases(
		s.NPOETC("-false -a -false -primary", "-primary", false),
		s.NPOETC("-false -false -primary", "-primary", false),
		s.NPOETC("-false -a -true -primary", "-primary", false),
		s.NPOETC("-false -true -primary", "-primary", false),
		s.NPOETC("-true -a -false -primary", "-primary", false),
		s.NPOETC("-true -false -primary", "-primary", false),
		s.NPOETC("-true -a -true -primary", "-primary", true),
		s.NPOETC("-true -true -primary", "-primary", true),
	)
}

func (s *PredicateExpressionTestSuite) TestOrOpEval() {
	s.RunTestCases(
		s.NPTC("-false -o -false -primary", "-primary", false),
		s.NPTC("-false -o -true -primary", "-primary", true),
		s.NPTC("-true -o -false -primary", "-primary", true),
		s.NPTC("-true -o -true -primary", "-primary", true),
	)
}

func (s *PredicateExpressionTestSuite) TestBinOpPrecedence() {
	s.RunTestCases(
		// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
		// Without precedence, this would be parsed as ((-true -o -true) -a false) which
		// evaluates to false.
		s.NPOETC("-true -o -true -a -false -primary", "-primary", true),
	)
}

func (s *PredicateExpressionTestSuite) TestParensErrors() {
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
		s.NPETC("( ( ( -true ) -a", "-a: no expression after -a"),
		s.NPETC("( ( ( -true ) -a ) )", "-a: no expression after -a"),
	)
}

func (s *PredicateExpressionTestSuite) TestParensEval() {
	s.RunTestCases(
		// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
		// which would evaluate to true.
		s.NPOETC("( -true -o -true ) -a -false -primary", "-primary", false),
		s.NPOETC("-not ( -true -o -false ) -primary", "-primary", false),
		s.NPOETC("( -true ) -a ( -false ) -primary", "-primary", false),
		s.NPOETC("( -true ( -false ) -o ( ( -false -true ) ) ) -primary", "-primary", false),
		s.NPOETC("( ( ( -true ) ) ) -primary", "-primary", true),
		s.NPOETC("( ( -true ) -a -false ) -primary", "-primary", false),
	)
}

func (s *PredicateExpressionTestSuite) TestComplexErrors() {
	s.RunTestCases(
		s.NPETC("( -true ) -a )", "): no beginning '('"),
		s.NPETC(".key -a -foo", "expected a predicate after key"),
	)
}

func (s *PredicateExpressionTestSuite) TestComplexOpEval() {
	s.RunTestCases(
		s.NPOETC("( -true -o -true ) -false -primary", "-primary", false),
		// Should be parsed as (-true -a -false) -o -true which evaluates to true.
		s.NPOETC("-true -false -o -true -primary", "-primary", true),
		s.NPOETC("-false -o -true -false -primary", "-primary", false),
		s.NPOETC("( -true -true ) -o -false -primary", "-primary", true),
	)
}

func (s *PredicateExpressionTestSuite) TestPredicateExpressions() {
	s.RunTestCases(
		// Parsed as "> 3 AND > 5"
		s.NPTC("+3 -a +5 -primary", "-primary", float64(6)),
		s.NPNTC("+3 -a +5 -primary", "-primary", float64(4)),
		// Parsed as "(> 3 AND > 5) OR == 1"
		s.NPTC("( +3 -a +5 ) -o 1 -primary", "-primary", float64(6)),
		s.NPNTC("+3 -a +5 -primary", "-primary", float64(4)),
		s.NPTC("( +3 -a +5 ) -o 1 -primary", "-primary", float64(1)),
		// Parsed as "NOT(> 3) OR > 5" which reduces to "<= 3 OR > 5"
		s.NPTC("-not +3 -o +5 -primary", "-primary", float64(3)),
		s.NPTC("-not +3 -o +5 -primary", "-primary", float64(1)),
		s.NPTC("-not +3 -o +5 -primary", "-primary", float64(6)),
		// Parsed as "true OR empty object/array"
		s.NPTC("-true -o -empty -primary", "-primary", true),
		s.NPTC("-true -o -empty -primary", "-primary", []interface{}{}),
		// Parsed as "true AND empty object/array" (should always return false)
		s.NPNTC("-true -a -empty -primary", "-primary", true),
		s.NPNTC("-true -a -empty -primary", "-primary", []interface{}{}),
	)
}

func TestPredicateExpression(t *testing.T) {
	s := new(PredicateExpressionTestSuite)
	s.Parser = newPredicateExpressionParser(false)
	suite.Run(t, s)
}