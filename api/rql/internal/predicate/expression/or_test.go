package expression

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestOr(t *testing.T) {
	s := new(BinOpTestSuite)
	s.opName = "OR"
	s.newOp = Or
	s.testEvalMethod = func(s *BinOpTestSuite, RFTC TCRunFunc, RTTC TCRunFunc, constructV func(string) interface{}) {
		p := Or(newMockP("1"), newMockP("2"))
		// p1 == false, p2 == false
		RFTC(s, p, constructV("3"))
		// false, true
		RTTC(s, p, constructV("2"))
		// true, false
		RTTC(s, p, constructV("1"))
		// true, true
		p.(*or).p2 = p.(*or).p1
		RTTC(s, p, constructV("1"))
	}
	suite.Run(t, s)
}
