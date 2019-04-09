package munge

import (
	"fmt"
	"testing"
	"time"

	"github.com/araddon/dateparse"
	"github.com/stretchr/testify/suite"
)

type TimeTestSuite struct {
	MungeTestSuite
}

func (suite *TimeTestSuite) TestToTime() {
	mustParse := func(v string) time.Time {
		t, err := dateparse.ParseAny(v)
		if err != nil {
			panic(fmt.Sprintf("Required %v to parse to a time.Time{} object but received error: %v", v, err))
		}
		return t
	}
	suite.mungeFunc = func(v interface{}) (interface{}, error) {
		return ToTime(v)
	}
	curTime := time.Now()
	suite.runTestCases(
		nTC(curTime, curTime),
		nTC(int64(-1), time.Time{}),
		nTC(int64(10000), time.Unix(10000, 0)),
		nETC(float64(10.5), "10.5.*decimal"),
		nTC(float64(-1), time.Time{}),
		nTC(float64(10000), time.Unix(10000, 0)),
		nTC("2006-01-02T15:04:05+7:00", mustParse("2006-01-02T15:04:05+7:00")),
		nTC("2006-01-02T15:04:05Z", mustParse("2006-01-02T15:04:05Z")),
		nETC("foo", "foo.*time.Time"),
	)
}

func TestToTime(t *testing.T) {
	suite.Run(t, new(TimeTestSuite))
}
