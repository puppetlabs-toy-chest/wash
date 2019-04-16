package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
}

func (suite *HelpersTestSuite) SetupSuite() {
	SetTestCache(datastore.NewMemCache())
}

func (suite *HelpersTestSuite) TearDownSuite() {
	UnsetTestCache()
}

func (suite *HelpersTestSuite) TestName() {
	e := NewEntry("foo")
	suite.Equal(Name(&e), "foo")
}

func (suite *HelpersTestSuite) TestCName() {
	e := NewEntry("foo/bar/baz")
	suite.Equal("foo#bar#baz", CName(&e))

	e.SetSlashReplacementChar(':')
	suite.Equal("foo:bar:baz", CName(&e))
}

func (suite *HelpersTestSuite) TestID() {
	e := NewEntry("foo/bar")

	suite.Panics(
		func() { ID(&e) },
		"plugin.ID: entry foo (cname foo#bar) has no ID",
	)
	e.setID("/foo/bar")
	suite.Equal(ID(&e), "/foo/bar")
}

type helpersTestsMockEntry struct {
	EntryBase
	mock.Mock
}

func newHelpersTestsMockEntry() *helpersTestsMockEntry {
	e := &helpersTestsMockEntry{
		EntryBase: NewEntry("mockEntry"),
	}
	e.SetTestID("id")
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
