package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/stretchr/testify/suite"
)

type PrimitivePredicateTestSuite struct {
	ParserTestSuite
}

func (suite *PrimitivePredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *PrimitivePredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *PrimitivePredicateTestSuite) TestErrors() {
	suite.runTestCases(
		// These cases ensure that parsePrimitivePredicate
		// returns a MatchError if it cannot parse a primitive
		// predicate
		nPETC("", "expected a primitive predicate", true),
		// These cases ensure that parsePrimitivePredicate
		// returns any parse errors found while parsing the
		// primitive predicates
		nPETC("--15", "positive.*number", false),
		nPETC("+{", ".*closing.*}", false),
	)
}

func (suite *PrimitivePredicateTestSuite) TestValidInput() {
	suite.runTestCases(
		nPTC("-null", "", nil),
		nPTC("-exists", "", "not nil"),
		nPTC("-true", "", true),
		nPTC("-false", "", false),
		nPTC("200", "", float64(200)),
		nPTC("+1h", "", addTST(-2*numeric.DurationOf('h'))),
		nPTC("+{1h}", "", addTST(2*numeric.DurationOf('h'))),
		nPTC("foo", "", "foo"),
		nPTC("+foo", "", "+foo"),
	)
}

func (suite *PrimitivePredicateTestSuite) TestNullP() {
	suite.True(nullP(nil))
	suite.False(nullP("not nil"))
}

func (suite *PrimitivePredicateTestSuite) TestExistsP() {
	suite.True(existsP("not nil"))
	suite.False(existsP(nil))
}

func (suite *PrimitivePredicateTestSuite) TestTrueP() {
	suite.False(trueP("foo"))
	suite.False(trueP(false))
	suite.True(trueP(true))
}

func (suite *PrimitivePredicateTestSuite) TestFalseP() {
	suite.False(falseP("foo"))
	suite.False(falseP(true))
	suite.True(falseP(false))
}

func TestPrimitivePredicate(t *testing.T) {
	s := new(PrimitivePredicateTestSuite)
	s.parser = parsePrimitivePredicate
	suite.Run(t, s)
}
