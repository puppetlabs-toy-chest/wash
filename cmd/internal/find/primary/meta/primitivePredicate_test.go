package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type PrimitivePredicateTestSuite struct {
	parsertest.Suite
}

func (s *PrimitivePredicateTestSuite) TestErrors() {
	s.RunTestCases(
		// These cases ensure that parsePrimitivePredicate
		// returns a MatchError if it cannot parse a primitive
		// predicate
		s.NPETC("", "expected a primitive predicate", true),
		// These cases ensure that parsePrimitivePredicate
		// returns any parse errors found while parsing the
		// primitive predicates
		s.NPETC("--15", "positive.*number", false),
		s.NPETC("+{", ".*closing.*}", false),
	)
}

func (s *PrimitivePredicateTestSuite) TestValidInput() {
	s.RunTestCases(
		s.NPTC("-null", "", nil),
		s.NPTC("-exists", "", "not nil"),
		s.NPTC("-true", "", true),
		s.NPTC("-false", "", false),
		s.NPTC("200", "", float64(200)),
		s.NPTC("+1h", "", addTST(-2*numeric.DurationOf('h'))),
		s.NPTC("+{1h}", "", addTST(2*numeric.DurationOf('h'))),
		s.NPTC("foo", "", "foo"),
		s.NPTC("+foo", "", "+foo"),
	)
}

func (s *PrimitivePredicateTestSuite) TestNullP() {
	s.True(nullP(nil))
	s.False(nullP("not nil"))
}

func (s *PrimitivePredicateTestSuite) TestExistsP() {
	s.True(existsP("not nil"))
	s.False(existsP(nil))
}

func (s *PrimitivePredicateTestSuite) TestTrueP() {
	s.False(trueP.IsSatisfiedBy("foo"))
	s.False(trueP.Negate().IsSatisfiedBy("foo"))

	s.False(trueP.IsSatisfiedBy(false))
	s.True(trueP.Negate().IsSatisfiedBy(false))

	s.True(trueP.IsSatisfiedBy(true))
	s.False(trueP.Negate().IsSatisfiedBy(true))
}

func (s *PrimitivePredicateTestSuite) TestFalseP() {
	s.False(falseP.IsSatisfiedBy("foo"))
	s.False(falseP.Negate().IsSatisfiedBy("foo"))

	s.False(falseP.IsSatisfiedBy(true))
	s.True(falseP.Negate().IsSatisfiedBy(true))

	s.True(falseP.IsSatisfiedBy(false))
	s.False(falseP.Negate().IsSatisfiedBy(false))
}

func TestPrimitivePredicate(t *testing.T) {
	s := new(PrimitivePredicateTestSuite)
	s.Parser = predicate.ToParser(parsePrimitivePredicate)
	suite.Run(t, s)
}
