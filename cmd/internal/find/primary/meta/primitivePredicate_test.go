package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type PrimitivePredicateTestSuite struct {
	parsertest.Suite
}

func (s *PrimitivePredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *PrimitivePredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
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
	s.False(trueP("foo"))
	s.False(trueP(false))
	s.True(trueP(true))
}

func (s *PrimitivePredicateTestSuite) TestFalseP() {
	s.False(falseP("foo"))
	s.False(falseP(true))
	s.True(falseP(false))
}

func TestPrimitivePredicate(t *testing.T) {
	s := new(PrimitivePredicateTestSuite)
	s.Parser = predicateParser(parsePrimitivePredicate)
	suite.Run(t, s)
}
