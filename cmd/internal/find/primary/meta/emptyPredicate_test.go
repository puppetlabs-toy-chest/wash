package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

type EmptyPredicateTestSuite struct {
	predicate.ParserTestSuite
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
	s.False(emptyP("foo"))
}

func (s *EmptyPredicateTestSuite) TestEmptyPObject() {
	mp := make(map[string]interface{})
	s.True(emptyP(mp))
	mp["foo"] = 1
	s.False(emptyP(mp))
}

func (s *EmptyPredicateTestSuite) TestEmptyPArray() {
	a := []interface{}{}
	s.True(emptyP(a))
	a = append(a, 1)
	s.False(emptyP(a))
}

func TestEmptyPredicate(t *testing.T) {
	s := new(EmptyPredicateTestSuite)
	s.Parser = predicate.GenericParser(parseEmptyPredicate)
	suite.Run(t, s)
}
