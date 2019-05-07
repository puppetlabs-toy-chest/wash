package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type EmptyPredicateTestSuite struct {
	parsertest.Suite
}

func (s *EmptyPredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "expected '-empty'", true),
		s.NPETC("foo", "expected '-empty'", true),
	)
}

func (s *EmptyPredicateTestSuite) TestValidInput() {
	s.RunTestCases(
		s.NPTC("-empty", "", []interface{}{}),
	)
}

func (s *EmptyPredicateTestSuite) TestEmptyPInvalidType() {
	p := emptyP(false)
	s.False(p.IsSatisfiedBy("foo"))
	s.False(p.Negate().IsSatisfiedBy("foo"))
}

func (s *EmptyPredicateTestSuite) TestEmptyPObject() {
	mp := make(map[string]interface{})
	p := emptyP(false)
	
	// Test empty map
	s.True(p.IsSatisfiedBy(mp))
	s.False(p.Negate().IsSatisfiedBy(mp))

	// Test nonempty map
	mp["foo"] = 1
	s.False(p.IsSatisfiedBy(mp))
	s.True(p.Negate().IsSatisfiedBy(mp))
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
	s.Parser = predicate.ToParser(parseEmptyPredicate)
	suite.Run(t, s)
}
