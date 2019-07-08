package meta

import (
	"regexp"
	"testing"

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
	parserTestSuite
}

func (s *PredicateExpressionTestSuite) RETC(input string, errRegex string) {
	s.Suite.RETC(input, regexp.QuoteMeta(errRegex), false)
}

// ROETC => RunOpEvalTestCase. input should only contain Boolean predicates (-true/-false).
// The Boolean predicates are meant to represent individual predicates that evaluate to -true/-false,
// respectively. Thus, ROETC("-false -o -false", "", false) is read as "Test that if p1 returns false,
// and p2 returns false, then p1 -o p2 returns false". It is _not_ meant to be confused with
// RTC("-false -o -false", "", false), which is read as "Test that the parsed predicate returns true if
// v is false OR is false". In other words, in RTC, "-false" is a Boolean predicate while in ROETC,
// "-false" represents _any_ predicate p (object/array/primitive/...) that returns false. ROETC is useful
// to make binary op/parens op tests more expressive.
func (s *PredicateExpressionTestSuite) ROETC(input string, remInput string, expectedValue bool) {
	// -false/-true will still be parsed as Boolean predicates. Using "true" as the satisfying value in
	// both cases ensures that -false/-true represent an arbitrary predicate p that returns false/true,
	// respectively (since -false(true) == false, -true(true) == true).
	if expectedValue {
		s.RTC(input, remInput, true)
	} else {
		s.RNTC(input, remInput, true)
	}
}

func (s *PredicateExpressionTestSuite) TestEmptyExpression() {
	s.RETC("", "expected a predicate expression")
	s.RETC("-primary", "unknown predicate -primary")
}

func (s *PredicateExpressionTestSuite) TestNotOpParseErrors() {
	// Test error when "-not" is supplied without an expression
	s.RETC("-not", "-not: no following expression")
	// Test error when "-not" is mixed with parentheses
	s.RETC("-not )", "): no beginning '('")
	s.RETC("( -not )", "-not: no following expression")
	// Test error when "-not" is supplied w/ a malformed predicate
	s.RETC("-not .", "expected a key sequence after '.'")
	// Test error when "-not" is followed by a binary operator
	s.RETC("-not -a", "-not: no following expression")
}

func (s *PredicateExpressionTestSuite) TestNotOpEval() {
	s.RTC("-not -true -primary", "-primary", false)
	s.RTC("-not -not -true -primary", "-primary", true)
	s.RTC("-not -not -not -true -primary", "-primary", false)
	// Ensure that "-not <predicate>" and "-not -not <predicate>" always
	// return false for mis-typed values. It is important to thoroughly
	// test this behavior because it can be very confusing for users to
	// debug if something goes wrong with the negation.
	//
	// Start with array predicates
	s.RNTC("-not [?] -true -primary", "-primary", "foo")
	s.RNTC("-not -not [?] -true -primary", "-primary", "foo")
	// empty predicate
	s.RNTC("-not -empty -primary", "-primary", "foo")
	s.RNTC("-not -not -empty -primary", "-primary", "foo")
	// numeric predicate
	s.RNTC("-not +1 -primary", "-primary", "2")
	s.RNTC("-not -not -1 -primary", "-primary", "0")
	// object predicate
	s.RNTC("-not .key -true -primary", "-primary", "foo")
	s.RNTC("-not -not .key -true -primary", "-primary", "foo")
	// boolean predicate
	s.RNTC("-not -true -primary", "-primary", "false")
	s.RNTC("-not -not -true -primary", "-primary", "true")
	// string predicate
	s.RNTC("-not f -primary", "-primary", 'g')
	s.RNTC("-not -not f -primary", "-primary", 'f')
	// time predicate
	s.RNTC("-not +1h -primary", "-primary", "not a time")
	s.RNTC("-not -not +1h -primary", "-primary", "not a time")
}

func (s *PredicateExpressionTestSuite) TestBinOpParseErrors() {
	// Tests for -and
	s.RETC("-a", "-a: no expression before -a")
	s.RETC("-true -a", "-a: no expression after -a")
	s.RETC("-true -a -a", "-a: no expression after -a")
	// Tests for -or
	s.RETC("-o", "-o: no expression before -o")
	s.RETC("-true -o", "-o: no expression after -o")
	s.RETC("-true -o -o", "-o: no expression after -o")
}

func (s *PredicateExpressionTestSuite) TestAndOpEval() {
	s.ROETC("-false -a -false -primary", "-primary", false)
	s.ROETC("-false -false -primary", "-primary", false)
	s.ROETC("-false -a -true -primary", "-primary", false)
	s.ROETC("-false -true -primary", "-primary", false)
	s.ROETC("-true -a -false -primary", "-primary", false)
	s.ROETC("-true -false -primary", "-primary", false)
	s.ROETC("-true -a -true -primary", "-primary", true)
	s.ROETC("-true -true -primary", "-primary", true)
}

func (s *PredicateExpressionTestSuite) TestOrOpEval() {
	s.RTC("-false -o -false -primary", "-primary", false)
	s.RTC("-false -o -true -primary", "-primary", true)
	s.RTC("-true -o -false -primary", "-primary", true)
	s.RTC("-true -o -true -primary", "-primary", true)
}

func (s *PredicateExpressionTestSuite) TestBinOpPrecedence() {
	// Should be parsed as (-true -o (-true -a -false)), which evaluates to true.
	// Without precedence, this would be parsed as ((-true -o -true) -a false) which
	// evaluates to false.
	s.ROETC("-true -o -true -a -false -primary", "-primary", true)
}

func (s *PredicateExpressionTestSuite) TestParensErrors() {
	// Test the simple error cases
	s.RETC(")", "): no beginning '('")
	s.RETC("(", "(: missing closing ')'")
	s.RETC("( )", "(): empty inner expression")
	// Test some more complicated error cases
	s.RETC("( -not", "-not: no following expression")
	s.RETC("( -true ) )", "): no beginning '('")
	s.RETC("( -true ) ( ) -true", "(): empty inner expression")
	s.RETC("( -true ( -false )", "(: missing closing ')'")
	s.RETC("( ( ( -true ) ) ) )", "): no beginning '('")
	s.RETC("( -a )", "-a: no expression before -a")
	s.RETC("( ( ( -true ) -a", "-a: no expression after -a")
	s.RETC("( ( ( -true ) -a ) )", "-a: no expression after -a")
}

func (s *PredicateExpressionTestSuite) TestParensEval() {
	// Note that w/o the parentheses, this would be parsed as "(-true -o (-true -a -false))"
	// which would evaluate to true.
	s.ROETC("( -true -o -true ) -a -false -primary", "-primary", false)
	s.ROETC("-not ( -true -o -false ) -primary", "-primary", false)
	s.ROETC("( -true ) -a ( -false ) -primary", "-primary", false)
	s.ROETC("( -true ( -false ) -o ( ( -false -true ) ) ) -primary", "-primary", false)
	s.ROETC("( ( ( -true ) ) ) -primary", "-primary", true)
	s.ROETC("( ( -true ) -a -false ) -primary", "-primary", false)
}

func (s *PredicateExpressionTestSuite) TestComplexErrors() {
	s.RETC("( -true ) -a )", "): no beginning '('")
	s.RETC(".key -a -foo", "expected a predicate after key")
}

func (s *PredicateExpressionTestSuite) TestComplexOpEval() {
	s.ROETC("( -true -o -true ) -false -primary", "-primary", false)
	// Should be parsed as (-true -a -false) -o -true which evaluates to true.
	s.ROETC("-true -false -o -true -primary", "-primary", true)
	s.ROETC("-false -o -true -false -primary", "-primary", false)
	s.ROETC("( -true -true ) -o -false -primary", "-primary", true)
}

func (s *PredicateExpressionTestSuite) TestPredicateExpressions() {
	// Parsed as "> 3 AND > 5"
	s.RTC("+3 -a +5 -primary", "-primary", float64(6))
	s.RNTC("+3 -a +5 -primary", "-primary", float64(4))
	// Parsed as "(> 3 AND > 5) OR == 1"
	s.RTC("( +3 -a +5 ) -o 1 -primary", "-primary", float64(6))
	s.RNTC("+3 -a +5 -primary", "-primary", float64(4))
	s.RTC("( +3 -a +5 ) -o 1 -primary", "-primary", float64(1))
	// Parsed as "NOT(> 3) OR > 5" which reduces to "<= 3 OR > 5"
	s.RTC("-not +3 -o +5 -primary", "-primary", float64(3))
	s.RTC("-not +3 -o +5 -primary", "-primary", float64(1))
	s.RTC("-not +3 -o +5 -primary", "-primary", float64(6))
	// Parsed as "true OR empty object/array"
	s.RTC("-true -o -empty -primary", "-primary", true)
	s.RTC("-true -o -empty -primary", "-primary", []interface{}{})
	// Parsed as "true AND empty object/array" (should always return false)
	s.RNTC("-true -a -empty -primary", "-primary", true)
	s.RNTC("-true -a -empty -primary", "-primary", []interface{}{})
}

func (s *PredicateExpressionTestSuite) TestDeMorgansLaw() {
	// Parsed as "NOT(3) AND NOT(6)"
	s.RTC("! ( 3 -o 6 ) -primary", "-primary", float64(5))
	s.RNTC("! ( 3 -o 6 ) -primary", "-primary", "foo")
	// Parsed as "NOT(3) OR NOT(6)"
	s.RTC("! ( 3 -a 6 ) -primary", "-primary", float64(3))
	s.RNTC("! ( 3 -a 6 ) -primary", "-primary", "foo")
}

func (s *PredicateExpressionTestSuite) TestDeMorgansLaw_SchemaP() {
	// Returns true if m['key1']['key2'] == primitive_value AND
	// m['key1'] == primitive_value, which is impossible.
	s.RNSTC("! .key1 ( .key2 3 -o 6 ) -primary", "-primary", ".key1.key2 p")
	s.RNSTC("! .key1 ( .key2 3 -o 6 ) -primary", "-primary", ".key1 p")
	// Returns true if m['key1']['key2'] == primitive_value OR
	// m['key1'] == primitive_value, which is possible.
	s.RSTC("! .key1 ( .key2 3 -a 6 ) -primary", "-primary", ".key1.key2 p")
	s.RSTC("! .key1 ( .key2 3 -a 6 ) -primary", "-primary", ".key1 p")
	// Returns true if []['key'] == primitive_value AND [] == primitive_value,
	// which is impossible.
	s.RNSTC("! [?] ( .key 3 -o 6 ) -primary", "-primary", "[].key p")
	s.RNSTC("! [?] ( .key 3 -o 6 ) -primary", "-primary", "[] p")
	// Returns true if []['key'] == primitive_value OR [] == primitive_value,
	// which is possible.
	s.RSTC("! [?] ( .key 3 -a 6 ) -primary", "-primary", "[].key p")
	s.RSTC("! [?] ( .key 3 -a 6 ) -primary", "-primary", "[] p")
}

func TestPredicateExpression(t *testing.T) {
	s := new(PredicateExpressionTestSuite)
	s.SetParser(newPredicateExpressionParser(false))
	suite.Run(t, s)
}
