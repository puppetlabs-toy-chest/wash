package expression

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestAnd(t *testing.T) {
	s := new(BinOpTestSuite)
	s.opName = "AND"
	s.newOp = And
	s.testEvalMethod = func(s *BinOpTestSuite, RFTC TCRunFunc, RTTC TCRunFunc, constructV func(string) interface{}) {
		p := And(newMockP("1"), newMockP("2"))
		// p1 == false, p2 == false
		RFTC(s, p, constructV("3"))
		// false, true
		RFTC(s, p, constructV("2"))
		// true, false
		RFTC(s, p, constructV("1"))
		// true, true
		p.(*and).p2 = p.(*and).p1
		RTTC(s, p, constructV("1"))
	}
	suite.Run(t, s)
}
