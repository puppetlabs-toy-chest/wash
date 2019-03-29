package plugin

import (
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type SyncableAttributeTestSuite struct {
	suite.Suite
}

func (suite *SyncableAttributeTestSuite) testAttr(
	attr SyncableAttribute,
	badValue interface{},
	goodValue interface{},
	mungedGoodValue interface{},
) {
	entry := NewEntry("foo")
	meta := EntryMetadata{}
	key := "bar"

	err := attr.sync(&entry, meta, key)
	suite.Regexp("does not.*bar.*", err)

	meta[key] = nil
	err = attr.sync(&entry, meta, key)
	suite.Regexp("bar.*null.*", err)

	meta[key] = badValue
	err = attr.sync(&entry, meta, key)
	suite.Regexp("munge.*bar.*key", err)

	meta[key] = goodValue
	err = attr.sync(&entry, meta, key)
	if suite.NoError(err) {
		expectedAttr := make(map[string]interface{})
		expectedAttr[attr.name] = mungedGoodValue
		expectedAttr["meta"] = EntryMetadata{}

		attr := entry.attributes()
		suite.Equal(expectedAttr, attr.toMap())
	}
}

func (suite *SyncableAttributeTestSuite) TestAtimeAttr() {
	t := time.Now()
	suite.testAttr(AtimeAttr(), "foo", t, t)
}

func (suite *SyncableAttributeTestSuite) TestMtimeAttr() {
	t := time.Now()
	suite.testAttr(MtimeAttr(), "foo", t, t)
}

func (suite *SyncableAttributeTestSuite) TestCtimeAttr() {
	t := time.Now()
	suite.testAttr(CtimeAttr(), "foo", t, t)
}

func (suite *SyncableAttributeTestSuite) TestModeAttr() {
	suite.testAttr(ModeAttr(), "badMode", "0777", os.FileMode(0777))
}

func (suite *SyncableAttributeTestSuite) TestSizeAttr() {
	suite.testAttr(SizeAttr(), "badSize", int64(12), uint64(12))
}

func (suite *SyncableAttributeTestSuite) TestMungeToTimeVal() {
	_, err := mungeToTimeVal("foo")
	suite.Regexp("parse.*foo", err)

	expectedTime := time.Now()
	t, err := mungeToTimeVal(expectedTime)
	if suite.NoError(err) {
		suite.Equal(expectedTime, t)
	}
}

func (suite *SyncableAttributeTestSuite) TestMungeToSizeVal() {
	type testCase struct {
		input    interface{}
		expected uint64
		errRegex string
	}

	cases := []testCase{
		{input: uint64(10), expected: 10},
		{input: int64(10), expected: 10},
		{input: int(10), expected: 10},
		{input: "foo", errRegex: "foo.*size.*uint64.*int.*int64"},
	}

	for _, c := range cases {
		actual, err := mungeToSizeVal(c.input)
		if c.errRegex != "" {
			suite.Regexp(regexp.MustCompile(c.errRegex), err)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, actual)
			}
		}
	}
}

func TestSyncableAttribute(t *testing.T) {
	suite.Run(t, new(SyncableAttributeTestSuite))
}
