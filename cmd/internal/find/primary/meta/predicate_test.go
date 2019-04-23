package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/stretchr/testify/suite"
)

type PredicateTestSuite struct {
	ParserTestSuite
}

func (suite *PredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (suite *PredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (suite *PredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", "expected either a primitive, object, or array predicate", true),
		// These cases ensure that parsePredicate returns any syntax errors found
		// while parsing the predicate
		nPETC(".", "expected a key sequence after '.'", false),
		nPETC("[", `expected a closing '\]'`, false),
		nPETC("--15", "positive", false),
	)
}

func (suite *PredicateTestSuite) TestValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	suite.runTestCases(
		// ObjectPredicate
		nPTC(".key -true", "", mp),
		// ArrayPredicate
		nPTC("[] -true", "", toA(true)),
		// PrimitivePredicate
		nPTC("-true", "", true),
	)
}

func TestPredicate(t *testing.T) {
	s := new(PredicateTestSuite)
	s.parser = parsePredicate
	suite.Run(t, s)
}
