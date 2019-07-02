package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type PrimitivePredicateTestSuite struct {
	parserTestSuite
}

func (s *PrimitivePredicateTestSuite) TestErrors() {
	// These cases ensure that parsePrimitivePredicate
	// returns a MatchError if it cannot parse a primitive
	// predicate
	s.RETC("", "expected a primitive predicate", true)
	// These cases ensure that parsePrimitivePredicate
	// returns any parse errors found while parsing the
	// primitive predicates
	s.RETC("--15", "positive.*number", false)
	s.RETC("+{", ".*closing.*}", false)
}

func (s *PrimitivePredicateTestSuite) TestValidInput() {
	s.RTC("-null", "", nil)
	s.RTC("-exists", "", "not nil")
	s.RTC("-true", "", true)
	s.RTC("-false", "", false)
	s.RTC("200", "", float64(200))
	s.RTC("+1h", "", addTRT(-2*numeric.DurationOf('h')))
	s.RTC("+{1h}", "", addTRT(2*numeric.DurationOf('h')))
	s.RTC("foo", "", "foo")
	s.RTC("+foo", "", "+foo")
}

func (s *PrimitivePredicateTestSuite) TestValidInput_SchemaP() {
	for _, input := range []string{"-null", "-true", "-false"} {
		s.RSTC(input, "", "p")
		s.RNSTC(input, "", "o")
		s.RNSTC(input, "", "a")
	}
	s.RSTC("-exists", "", "p")
}

func (s *PrimitivePredicateTestSuite) TestNullP() {
	s.True(nullP().IsSatisfiedBy(nil))
	s.False(nullP().IsSatisfiedBy("not nil"))
	s.True(nullP().schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func (s *PrimitivePredicateTestSuite) TestExistsP() {
	s.True(existsP().IsSatisfiedBy("not nil"))
	s.False(existsP().IsSatisfiedBy(nil))
}

func (s *PrimitivePredicateTestSuite) TestsExistsP_SchemaP() {
	ep := existsP()
	nep := ep.Negate().(Predicate)

	s.True(ep.schemaP().IsSatisfiedBy(s.newSchema("p")))
	s.True(ep.schemaP().IsSatisfiedBy(s.newSchema("o")))
	s.True(ep.schemaP().IsSatisfiedBy(s.newSchema("a")))

	s.False(nep.schemaP().IsSatisfiedBy(s.newSchema("p")))
	s.False(nep.schemaP().IsSatisfiedBy(s.newSchema("o")))
	s.False(nep.schemaP().IsSatisfiedBy(s.newSchema("a")))
}

func (s *PrimitivePredicateTestSuite) TestTrueP() {
	negatedTrueP := trueP().Negate().(Predicate)

	s.False(trueP().IsSatisfiedBy("foo"))
	s.False(negatedTrueP.IsSatisfiedBy("foo"))

	s.False(trueP().IsSatisfiedBy(false))
	s.True(negatedTrueP.IsSatisfiedBy(false))

	s.True(trueP().IsSatisfiedBy(true))
	s.False(negatedTrueP.IsSatisfiedBy(true))

	s.True(trueP().schemaP().IsSatisfiedBy(s.newSchema("p")))
	s.True(negatedTrueP.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func (s *PrimitivePredicateTestSuite) TestFalseP() {
	negatedFalseP := falseP().Negate().(Predicate)

	s.False(falseP().IsSatisfiedBy("foo"))
	s.False(negatedFalseP.IsSatisfiedBy("foo"))

	s.False(falseP().IsSatisfiedBy(true))
	s.True(negatedFalseP.IsSatisfiedBy(true))

	s.True(falseP().IsSatisfiedBy(false))
	s.False(negatedFalseP.IsSatisfiedBy(false))

	s.True(falseP().schemaP().IsSatisfiedBy(s.newSchema("p")))
	s.True(negatedFalseP.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func TestPrimitivePredicate(t *testing.T) {
	s := new(PrimitivePredicateTestSuite)
	s.SetParser(predicate.ToParser(parsePrimitivePredicate))
	suite.Run(t, s)
}
