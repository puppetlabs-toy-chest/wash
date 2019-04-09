package munge

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SizeTestSuite struct {
	MungeTestSuite
}

func (suite *SizeTestSuite) TestToSize() {
	suite.mungeFunc = func(v interface{}) (interface{}, error) {
		return ToSize(v)
	}
	suite.runTestCases(
		nTC(uint64(10), uint64(10)),
		nETC(int(-1), "-1.*negative.*size"),
		nTC(int(10), uint64(10)),
		nTC(int32(10), uint64(10)),
		nTC(int64(10), uint64(10)),
		nETC(float64(10.5), "10.5.*decimal.*size"),
		nTC(float64(10.0), uint64(10)),
		nETC("foo", "foo.*valid.*size.*uint64.*int.*int32.*int64.*float64"),
	)
}

func TestToSize(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}
