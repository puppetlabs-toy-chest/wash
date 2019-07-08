package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type TimePredicateTestSuite struct {
	parserTestSuite
}

func (s *TimePredicateTestSuite) TestErrors() {
	s.RETC("", `expected a \+, -, or a digit`, true)
	s.RETC("200", "expected a duration", true)
	s.RETC("+{", ".*closing.*}", false)
}

func (s *TimePredicateTestSuite) TestValidInputTrueValues() {
	// Test the happy cases first
	s.RTC("+2h -size", "-size", addTRT(-3*numeric.DurationOf('h')))
	s.RTC("-2h -size", "-size", addTRT(-1*numeric.DurationOf('h')))
	s.RTC("+{2h} -size", "-size", addTRT(3*numeric.DurationOf('h')))
	s.RTC("-{2h} -size", "-size", addTRT(1*numeric.DurationOf('h')))
	// Test a stringified time to ensure that munge.ToTime's called
	s.RTC("+2h -size", "-size", addTRT(-3*numeric.DurationOf('h')).String())
}

func (s *TimePredicateTestSuite) TestValidInputFalseValues() {
	s.RNTC("+2h", "", "not_a_valid_time_value")
	s.RNTC("+2h", "", addTRT(-1*numeric.DurationOf('h')))
	s.RNTC("-2h", "", addTRT(-3*numeric.DurationOf('h')))
	s.RNTC("+{2h}", "", addTRT(1*numeric.DurationOf('h')))
	s.RNTC("-{2h}", "", addTRT(3*numeric.DurationOf('h')))
	// Test time mis-matches. First case is a future/past mismatch,
	// while the second case is a past/future mismatch.
	s.RNTC("-{2h}", "", addTRT(-5*numeric.DurationOf('h')))
	s.RNTC("-2h", "", addTRT(5*numeric.DurationOf('h')))
}

func (s *TimePredicateTestSuite) TestValidInput_SchemaP() {
	s.RSTC("2h", "", "p")
	s.RNSTC("2h", "", "o")
	s.RNSTC("2h", "", "a")
}

func (s *TimePredicateTestSuite) TestTimeP_Negation_NotATime() {
	d := 5 * numeric.DurationOf('h')
	tp := timeP(true, func(n int64) bool {
		return n > d
	})
	ntp := tp.Negate().(Predicate)
	s.False(ntp.IsSatisfiedBy("not_a_valid_time_value"))
	// The schemaP should still return true for a primitive value
	s.True(ntp.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func (s *TimePredicateTestSuite) TestTimeP_Negation_TimeMismatch() {
	d := 5 * numeric.DurationOf('h')
	// These tests check that negating a timePredicate will still return
	// false for time mismatches.

	// Test past queries
	tp := timeP(true, func(n int64) bool {
		return n > d
	})
	s.False(tp.Negate().IsSatisfiedBy(addTRT(d + 1)))
	s.False(tp.Negate().IsSatisfiedBy(addTRT(d - 1)))

	// Test future queries
	tp = timeP(false, func(n int64) bool {
		return n > d
	})
	s.False(tp.Negate().IsSatisfiedBy(addTRT(-(d + 1))))
	s.False(tp.Negate().IsSatisfiedBy(addTRT(-(d - 1))))
}

func (s *TimePredicateTestSuite) TestTimeP_Negation() {
	d := 5 * numeric.DurationOf('h')
	tp := timeP(true, func(n int64) bool {
		return n > d
	})
	ntp := tp.Negate().(Predicate)
	s.False(ntp.IsSatisfiedBy(addTRT(-(d + 1))))
	s.True(ntp.IsSatisfiedBy(addTRT(-(d - 1))))
	// The schemaP should still return true for a primitive value
	s.True(ntp.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

// addTRT => addToReferenceTime. Saves some typing. Note that v
// is an int64 duration.
func addTRT(v int64) time.Time {
	return params.ReferenceTime.Add(time.Duration(v))
}

func TestTimePredicate(t *testing.T) {
	s := new(TimePredicateTestSuite)
	s.SetParser(predicate.ToParser(parseTimePredicate))
	suite.Run(t, s)
}
