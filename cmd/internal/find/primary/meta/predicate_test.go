package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type PredicateTestSuite struct {
	parsertest.Suite
}

func (s *PredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *PredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (s *PredicateTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "expected either a primitive, object, or array predicate", true),
		// These cases ensure that parsePredicate returns any syntax errors found
		// while parsing the predicate
		s.NPETC(".", "expected a key sequence after '.'", false),
		s.NPETC("[", `expected a closing '\]'`, false),
		s.NPETC("--15", "positive", false),
	)
}

func (s *PredicateTestSuite) TestValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	s.RunTestCases(
		// ObjectPredicate
		s.NPTC(".key -true", "", mp),
		// ArrayPredicate
		s.NPTC("[?] -true", "", toA(true)),
		// PrimitivePredicate
		s.NPTC("-true", "", true),
	)
}

func TestPredicate(t *testing.T) {
	s := new(PredicateTestSuite)
	s.Parser = predicate.GenericParser(parsePredicate)
	suite.Run(t, s)
}
