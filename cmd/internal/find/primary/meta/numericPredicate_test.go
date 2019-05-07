package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type NumericPredicateTestSuite struct {
	parsertest.Suite
}

func (s *NumericPredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", `expected a \+, -, or a digit`, true),
		s.NPETC("foo", "expected.*number.*foo", true),
		s.NPETC("--15", "expected.*positive", false),
	)
}

func (s *NumericPredicateTestSuite) TestValidInput() {
	// Test the happy cases first
	s.RunTestCases(
		// Test a plain numeric value
		s.NPTC("200 -size", "-size", float64(200)),
		s.NPTC("+200 -size", "-size", float64(201)),
		s.NPTC("-200 -size", "-size", float64(199)),
		// Test a plain, negative numeric value
		s.NPTC("{200} -size", "-size", float64(-200)),
		s.NPTC("+{200} -size", "-size", float64(-199)),
		s.NPTC("-{200} -size", "-size", float64(-201)),
		// Test a size value
		s.NPTC("2G -size", "-size", float64(2*numeric.BytesOf('G'))),
		s.NPTC("+2G -size", "-size", float64(3*numeric.BytesOf('G'))),
		s.NPTC("-2G -size", "-size", float64(1*numeric.BytesOf('G'))),
	)

	// Now test that the predicate returns false for a non-numeric
	// value (i.e. a non float64 value)
	p, _, err := parseNumericPredicate(s.ToTks("200"))
	if s.NoError(err) {
		s.False(p("200"))
	}
}

func TestNumericPredicate(t *testing.T) {
	s := new(NumericPredicateTestSuite)
	s.Parser = predicateParser(parseNumericPredicate)
	suite.Run(t, s)
}
