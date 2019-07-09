package munge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ModeTestSuite struct {
	MungeTestSuite
}

func (suite *ModeTestSuite) TestParseMode() {
	suite.mungeFunc = func(v interface{}) (interface{}, error) {
		return parseMode(v)
	}
	suite.runTestCases(
		nTC(uint64(10), uint64(10)),
		nTC(int64(10), uint64(10)),
		nTC(float64(10.0), uint64(10)),
		nETC(float64(10.5), "decimal.*number"),
		nETC([]byte("invalid mode type"), "uint64.*int64.*float64.*string"),
		nTC("15", uint64(15)),
		nTC("0777", uint64(511)),
		nTC("0xf", uint64(15)),
		nETC("not a number", "not a number"),
	)
}

func (suite *ModeTestSuite) TestToFileMode() {
	// toFM saves having to type os.FileMode(v) when creating the
	// test cases
	toFM := func(v os.FileMode) os.FileMode { return v }
	suite.mungeFunc = func(v interface{}) (interface{}, error) {
		return ToFileMode(v)
	}
	suite.runTestCases(
		nTC(os.FileMode(0777), toFM(0777)),
		nETC("not a number", "not a number"),
		// 16877 is 0x41ed in decimal
		nTC("0x41ed", toFM(0755|os.ModeDir)),
		nTC(float64(16877), toFM(0755|os.ModeDir)),
		// 33188 is 0x81a4 in decimal
		nTC("0x81a4", toFM(0644)),
		nTC(float64(33188), toFM(0644)),
		nTC("0x21b6", toFM(0666|os.ModeCharDevice)),
	)
}

func TestMode(t *testing.T) {
	suite.Run(t, new(ModeTestSuite))
}
