package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type TimePredicateTestSuite struct {
	parsertest.Suite
}

func (s *TimePredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *TimePredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (s *TimePredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", `expected a \+, -, or a digit`, true),
		s.NPETC("200", "expected a duration", true),
		s.NPETC("+{", ".*closing.*}", false),
	)
}

func (s *TimePredicateTestSuite) TestValidInputTrueValues() {
	// Test the happy cases first
	s.RunTestCases(
		s.NPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h'))),
		s.NPTC("-2h -size", "-size", addTST(-1*numeric.DurationOf('h'))),
		s.NPTC("+{2h} -size", "-size", addTST(3*numeric.DurationOf('h'))),
		s.NPTC("-{2h} -size", "-size", addTST(1*numeric.DurationOf('h'))),
		// Test a stringified time to ensure that munge.ToTime's called
		s.NPTC("+2h -size", "-size", addTST(-3*numeric.DurationOf('h')).String()),
	)
}

func (s *TimePredicateTestSuite) TestValidInputFalseValues() {
	s.RunTestCases(
		s.NPNTC("+2h", "", "not_a_valid_time_value"),
		s.NPNTC("+2h", "", addTST(-1*numeric.DurationOf('h'))),
		s.NPNTC("-2h", "", addTST(-3*numeric.DurationOf('h'))),
		s.NPNTC("+{2h}", "", addTST(1*numeric.DurationOf('h'))),
		s.NPNTC("-{2h}", "", addTST(3*numeric.DurationOf('h'))),
		// Test time mis-matches. First case is a future/past mismatch,
		// while the second case is a past/future mismatch.
		s.NPNTC("-{2h}", "", addTST(-5*numeric.DurationOf('h'))),
		s.NPNTC("-2h", "", addTST(5*numeric.DurationOf('h'))),
	)
}

func (s *TimePredicateTestSuite) TestTimeP_Negation_NotATime() {
	d := 5 * numeric.DurationOf('h')
	tp := timeP(true, func(n int64) bool {
		return n > d 
	})
	s.False(tp.Negate().IsSatisfiedBy("not_a_valid_time_value"))
}

func (s *TimePredicateTestSuite) TestTimeP_Negation_TimeMismatch() {
	d := 5 * numeric.DurationOf('h')
	// This is a query to the past
	tp := timeP(true, func(n int64) bool {
		return n > d 
	})
	/// Negating tp still preserves time mismatches. 
	s.False(tp.Negate().IsSatisfiedBy(addTST(d+1)))
	s.False(tp.Negate().IsSatisfiedBy(addTST(d-1)))
}

func (s *TimePredicateTestSuite) TestTimeP_Negation() {
	d := 5 * numeric.DurationOf('h')
	tp := timeP(true, func(n int64) bool {
		return n > d 
	})
	s.False(tp.Negate().IsSatisfiedBy(addTST(-(d+1))))
	s.True(tp.Negate().IsSatisfiedBy(addTST(-(d-1))))
}

// addTST => addToStartTime. Saves some typing. Note that v
// is an int64 duration.
func addTST(v int64) time.Time {
	return params.StartTime.Add(time.Duration(v))
}

func TestTimePredicate(t *testing.T) {
	s := new(TimePredicateTestSuite)
	s.Parser = predicate.ToParser(parseTimePredicate)
	suite.Run(t, s)
}
