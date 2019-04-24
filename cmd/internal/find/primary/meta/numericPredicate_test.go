package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type NumericPredicateTestSuite struct {
	ParserTestSuite
}

func (suite *NumericPredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", `expected a \+, -, or a digit`, true),
		nPETC("foo", "expected.*number.*foo", true),
		nPETC("--15", "expected.*positive", false),
	)
}

func (suite *NumericPredicateTestSuite) TestValidInput() {
	// Test the happy cases first
	suite.runTestCases(
		// Test a plain numeric value
		nPTC("200 -size", "-size", float64(200)),
		nPTC("+200 -size", "-size", float64(201)),
		nPTC("-200 -size", "-size", float64(199)),
		// Test a plain, negative numeric value
		nPTC("{200} -size", "-size", float64(-200)),
		nPTC("+{200} -size", "-size", float64(-199)),
		nPTC("-{200} -size", "-size", float64(-201)),
		// Test a size value
		nPTC("2G -size", "-size", float64(2*numeric.BytesOf('G'))),
		nPTC("+2G -size", "-size", float64(3*numeric.BytesOf('G'))),
		nPTC("-2G -size", "-size", float64(1*numeric.BytesOf('G'))),
	)

	// Now test that the predicate returns false for a non-numeric
	// value (i.e. a non float64 value)
	p, _, err := parseNumericPredicate(toTks("200"))
	if suite.NoError(err) {
		suite.False(p("200"))
	}
}

func TestNumericPredicate(t *testing.T) {
	s := new(NumericPredicateTestSuite)
	s.parser = parseNumericPredicate
	suite.Run(t, s)
}
