package meta

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type PredicateTestSuite struct {
	parsertest.Suite
}

func (s *PredicateTestSuite) TestErrors() {
	s.RETC("", "expected either a primitive, object, or array predicate", true)
	// These cases ensure that parsePredicate returns any syntax errors found
	// while parsing the predicate
	s.RETC(".", "expected a key sequence after '.'", false)
	s.RETC("[", `expected a closing '\]'`, false)
	s.RETC("--15", "positive", false)
	// These cases ensure that parsePredicate does not parse any expression operators.
	// Otherwise, parsePredicateExpression may not work correctly.
	s.RETC("-a", ".*primitive.*", true)
	s.RETC("-and", ".*primitive.*", true)
	s.RETC("-o", ".*primitive.*", true)
	s.RETC("-or", ".*primitive.*", true)
	s.RETC("!", ".*primitive.*", true)
	s.RETC("-not", ".*primitive.*", true)
	s.RETC("(", ".*primitive.*", true)
	s.RETC(")", ".*primitive.*", true)
}

func (s *PredicateTestSuite) TestValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	// ObjectPredicate
	s.RTC(".key -true", "", mp)
	// ArrayPredicate
	s.RTC("[?] -true", "", toA(true))
	// PrimitivePredicate
	s.RTC("-true", "", true)
}

func TestPredicate(t *testing.T) {
	s := new(PredicateTestSuite)
	s.Parser = predicate.ToParser(parsePredicate)
	suite.Run(t, s)
}
