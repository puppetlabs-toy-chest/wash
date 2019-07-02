package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmptyPredicateTestSuite struct {
	parserTestSuite
}

func (s *EmptyPredicateTestSuite) TestErrors() {
	s.RETC("", "expected '-empty'", true)
	s.RETC("foo", "expected '-empty'", true)
}

func (s *EmptyPredicateTestSuite) TestValidInput() {
	s.RTC("-empty", "", []interface{}{})
}

func (s *EmptyPredicateTestSuite) TestValidInput_SchemaP() {
	s.RSTC("-empty", "", "o")
	s.RSTC("-empty", "", "a")
	s.RNSTC("-empty", "", "p")
}

func (s *EmptyPredicateTestSuite) TestEmptyPInvalidType() {
	p := emptyP(false)
	s.False(p.IsSatisfiedBy("foo"))
	s.False(p.Negate().IsSatisfiedBy("foo"))
}

func (s *EmptyPredicateTestSuite) TestEmptyPObject() {
	mp := make(map[string]interface{})
	p := emptyP(false)
	np := p.Negate().(Predicate)

	// Test empty map
	s.True(p.IsSatisfiedBy(mp))
	s.False(np.IsSatisfiedBy(mp))

	// Test nonempty map
	mp["foo"] = 1
	s.False(p.IsSatisfiedBy(mp))
	s.True(np.IsSatisfiedBy(mp))

	// Test the schemaP
	s.True(p.schemaP().IsSatisfiedBy(s.newSchema("o")))
	s.True(p.schemaP().IsSatisfiedBy(s.newSchema("a")))
	s.True(np.schemaP().IsSatisfiedBy(s.newSchema("o")))
	s.True(np.schemaP().IsSatisfiedBy(s.newSchema("a")))
}

func (s *EmptyPredicateTestSuite) TestEmptyPArray() {
	a := []interface{}{}
	p := emptyP(false)

	// Test empty array
	s.True(p.IsSatisfiedBy(a))
	s.False(p.Negate().IsSatisfiedBy(a))

	// Test nonempty array
	a = append(a, 1)
	s.False(p.IsSatisfiedBy(a))
	s.True(p.Negate().IsSatisfiedBy(a))
}

func TestEmptyPredicate(t *testing.T) {
	s := new(EmptyPredicateTestSuite)
	s.SetParser(toPredicateParser(parseEmptyPredicate))
	suite.Run(t, s)
}
