package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

func TestAnd(t *testing.T) {
	s := new(BinOpTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return And(newMockP(""), newMockP(""))
	}
	s.opName = "AND"
	s.testEvalMethod = func(s *BinOpTestSuite, RFTC TCRunFunc, RTTC TCRunFunc, constructV func(string) interface{}) {
		ast := s.A("AND", "1", "2")
		// p1 == false, p2 == false
		RFTC(s, ast, constructV("3"))
		// false, true
		RFTC(s, ast, constructV("2"))
		// true, false
		RFTC(s, ast, constructV("1"))
		// true, true
		ast = s.A("AND", "1", "1")
		RTTC(s, ast, constructV("1"))
	}
	suite.Run(t, s)
}
