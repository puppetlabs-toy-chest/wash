package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type StringPredicateTestSuite struct {
	parserTestSuite
}

func (s *StringPredicateTestSuite) TestErrors() {
	s.RETC("", "expected a nonempty string", true)

	_, _, err := parseStringPredicate([]string{""})
	s.Regexp("expected a nonempty string", err)
}

func (s *StringPredicateTestSuite) TestValidInput() {
	s.RTC("foo -size", "-size", "foo")
	s.RNTC("foo -size", "-size", "bar")
}

func (s *StringPredicateTestSuite) TestValidInput_SchemaP() {
	s.RSTC("foo", "", "p")
	s.RNSTC("foo", "", "o")
	s.RNSTC("foo", "", "a")
}

func (s *StringPredicateTestSuite) TestStringP_NotAString() {
	sp := stringP(func(s string) bool {
		return s == "f"
	})

	s.False(sp.IsSatisfiedBy('f'))
	nsp := sp.Negate().(Predicate)
	s.False(nsp.IsSatisfiedBy('g'))
	// The schemaP should still return true for a primitive value
	s.True(nsp.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func (s *StringPredicateTestSuite) TestStringP() {
	sp := stringP(func(s string) bool {
		return s == "f"
	})

	s.True(sp.IsSatisfiedBy("f"))

	// Test negation
	nsp := sp.Negate().(Predicate)
	s.False(nsp.IsSatisfiedBy("f"))
	s.True(nsp.IsSatisfiedBy("g"))
	s.True(nsp.schemaP().IsSatisfiedBy(s.newSchema("p")))
}

func TestStringPredicate(t *testing.T) {
	s := new(StringPredicateTestSuite)
	s.SetParser(predicate.ToParser(parseStringPredicate))
	suite.Run(t, s)
}
