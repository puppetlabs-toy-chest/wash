package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/stretchr/testify/suite"
)

type ArrayPredicateTestSuite struct {
	ParserTestSuite
}

func (suite *ArrayPredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *ArrayPredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *ArrayPredicateTestSuite) TestParseArrayPredicateErrors() {
	suite.runTestCases(
		nPETC("", `expected an opening '\['`, true),
		nPETC("]", `expected an opening '\['`, false),
		nPETC("f", `expected an opening '\['`, true),
		nPETC("[", `expected a closing '\]'`, false),
		nPETC("[a", `expected a closing '\]'`, false),
		nPETC("[*a]", `expected a closing '\]' after '\*'`, false),
		nPETC("[a]", `expected an array index inside '\[\]'`, false),
		nPETC("[-15]", `expected an array index inside '\[\]'`, false),
		nPETC("[]-true", `expected a '\.' or '\[' after '\]' but got -true instead`, false),
		nPETC("[]", `expected a predicate after \[\]`, false),
		nPETC("[*]", `expected a predicate after \[\*\]`, false),
		nPETC("[15]", `expected a predicate after \[15\]`, false),
		nPETC("[] +{", "expected.*closing.*}", false),
	)
}

func (suite *ArrayPredicateTestSuite) TestParseArrayPredicateValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	suite.runTestCases(
		// Test -empty
		nPTC("-empty", "", []interface{}{}),
		// Test each of the possible arrayPs
		nPTC("[] -true -size", "-size", toA(false, true)),
		nPTC("[*] -true -size", "-size", toA(true, true)),
		nPTC("[0] -true -size", "-size", toA(true)),
		// Test key sequences
		nPTC("[][] -true -size", "-size", toA(toA(true))),
		nPTC("[].key -true -size", "-size", toA(mp)),
	)
}

func (suite *ArrayPredicateTestSuite) TestParseArrayPredicateType() {
	// These test only the valid inputs. The error cases are tested in
	// TestParseArrayPredicateErrors.

	ptype, token, err := parseArrayPredicateType("[]")
	if suite.NoError(err) {
		suite.Equal(byte('s'), ptype.t)
		suite.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[*]")
	if suite.NoError(err) {
		suite.Equal(byte('a'), ptype.t)
		suite.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[15]")
	if suite.NoError(err) {
		suite.Equal(byte('n'), ptype.t)
		suite.Equal(uint(15), ptype.n)
		suite.Equal("", token)
	}
}

func (suite *ArrayPredicateTestSuite) TestArrayPSome() {
	p := arrayP(arrayPredicateType{t: 's'}, trueP)

	// Returns false for a non-array value
	suite.False(p("foo"))

	// Returns false if no elements satisfy p
	suite.False(p(toA(false)))

	// Returns true if some element satifies p
	suite.True(p(toA(true)))
	suite.True(p(toA("foo", true)))
}

func (suite *ArrayPredicateTestSuite) TestArrayPAll() {
	p := arrayP(arrayPredicateType{t: 'a'}, trueP)

	// Returns false for a non-array value
	suite.False(p("foo"))

	// Returns false if no elements satisfy p
	suite.False(p(toA(false)))

	// Returns false if only some of the elements satisfy p
	suite.False(p(toA(false, true)))

	// Returns true if all the elements satisfy P
	suite.True(p(toA(true)))
	suite.True(p(toA(true, true)))
}

func (suite *ArrayPredicateTestSuite) TestArrayPNth() {
	p := arrayP(arrayPredicateType{t: 'n', n: 1}, trueP)

	// Returns false for a non-array value
	suite.False(p("foo"))

	// Returns false if n >= len(vs)
	suite.False(p(toA(true)))

	// Returns false if vs[n] does not satisfy p
	suite.False(p(toA(true, false)))

	// Returns true if vs[n] satisfies p
	suite.True(p(toA(true, true)))
}

func toA(vs ...interface{}) []interface{} {
	return vs
}

func TestArrayPredicate(t *testing.T) {
	s := new(ArrayPredicateTestSuite)
	s.parser = parseArrayPredicate
	suite.Run(t, s)
}
