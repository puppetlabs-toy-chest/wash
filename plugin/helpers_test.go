package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
	root Entry
}

func (suite *HelpersTestSuite) SetupSuite() {
	SetTestCache(datastore.NewMemCache())
	root := NewRootEntry("root")
	suite.root = &root
}

func (suite *HelpersTestSuite) TearDownSuite() {
	UnsetTestCache()
}

func (suite *HelpersTestSuite) TestName() {
	e := suite.root.NewEntry("foo")
	suite.Equal(Name(&e), "foo")
}

func (suite *HelpersTestSuite) TestCName() {
	e := suite.root.NewEntry("foo/bar/baz")
	suite.Equal("foo#bar#baz", CName(&e))

	e.SetSlashReplacementChar(':')
	suite.Equal("foo:bar:baz", CName(&e))
}

func (suite *HelpersTestSuite) TestPath() {
	e := suite.root.NewEntry("bar")

	suite.Equal(Path(&e), "/root/bar")
}

func (suite *HelpersTestSuite) TestParseMode() {
	type testCase struct {
		input    interface{}
		expected uint64
		errRegex string
	}

	cases := []testCase{
		{input: uint64(10), expected: 10},
		{input: int64(10), expected: 10},
		{input: float64(10.0), expected: 10},
		{input: float64(10.5), errRegex: "decimal.*number"},
		{input: []byte("invalid mode type"), errRegex: "uint64.*int64.*float64.*string"},
		{input: "15", expected: 15},
		{input: "0777", expected: 511},
		{input: "0xf", expected: 15},
		{input: "not a number", errRegex: "not a number"},
	}

	for _, c := range cases {
		actual, err := parseMode(c.input)
		if c.errRegex != "" {
			suite.Regexp(regexp.MustCompile(c.errRegex), err)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, actual)
			}
		}
	}
}

func (suite *HelpersTestSuite) TestToFileMode() {
	type testCase struct {
		input    interface{}
		expected os.FileMode
		errRegex string
	}

	cases := []testCase{
		{input: "not a number", errRegex: "not a number"},
		// 16877 is 0x41ed in decimal
		{input: "0x41ed", expected: 0755 | os.ModeDir},
		{input: float64(16877), expected: 0755 | os.ModeDir},
		// 33188 is 0x81a4 in decimal
		{input: "0x81a4", expected: 0644},
		{input: float64(33188), expected: 0644},
	}

	for _, c := range cases {
		actual, err := ToFileMode(c.input)
		if c.errRegex != "" {
			suite.Regexp(regexp.MustCompile(c.errRegex), err)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, actual)
			}
		}
	}
}

type helpersTestsMockEntry struct {
	EntryBase
	mock.Mock
}

func newHelpersTestsMockEntry() *helpersTestsMockEntry {
	e := &helpersTestsMockEntry{
		EntryBase: NewRootEntry("mockEntry"),
	}
	e.DisableDefaultCaching()

	return e
}

func (suite *HelpersTestSuite) TestAttributes() {
	e := newHelpersTestsMockEntry()
	e.attr = EntryAttributes{}
	e.attr.SetCtime(time.Now())
	suite.Equal(e.attr, Attributes(e))
}

func (suite *HelpersTestSuite) TestExitCodeFromErr() {
	exitCode, err := ExitCodeFromErr(nil)
	if suite.NoError(err) {
		suite.Equal(
			0,
			exitCode,
			"ExitCodeFromErr should return an exit code of 0 if no error was passed-in",
		)
	}

	arbitraryErr := fmt.Errorf("an arbitrary error")
	_, err = ExitCodeFromErr(arbitraryErr)
	suite.EqualError(err, arbitraryErr.Error())

	// The default exit code is 0 for an empty ProcessState object
	exitErr := &exec.ExitError{ProcessState: &os.ProcessState{}}
	exitCode, err = ExitCodeFromErr(exitErr)
	if suite.NoError(err) {
		suite.Equal(
			0,
			exitCode,
			"ExitCodeFromErr should return the ExitError's exit code",
		)
	}
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
