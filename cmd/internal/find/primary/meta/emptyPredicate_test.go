package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EmptyPredicateTestSuite struct {
	ParserTestSuite
}

func (suite *EmptyPredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", "expected '-empty'", true),
		nPETC("foo", "expected '-empty'", true),
	)
}

func (suite *EmptyPredicateTestSuite) TestValidInput() {
	suite.runTestCases(
		nPTC("-empty", "", []interface{}{}),
	)
}

func (suite *EmptyPredicateTestSuite) TestEmptyPInvalidType() {
	suite.False(emptyP("foo"))
}

func (suite *EmptyPredicateTestSuite) TestEmptyPObject() {
	mp := make(map[string]interface{})
	suite.True(emptyP(mp))
	mp["foo"] = 1
	suite.False(emptyP(mp))
}

func (suite *EmptyPredicateTestSuite) TestEmptyPArray() {
	a := []interface{}{}
	suite.True(emptyP(a))
	a = append(a, 1)
	suite.False(emptyP(a))
}

func TestEmptyPredicate(t *testing.T) {
	s := new(EmptyPredicateTestSuite)
	s.parser = parseEmptyPredicate
	suite.Run(t, s)
}
