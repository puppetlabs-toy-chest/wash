package plugin

import (
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
	e := NewEntryBase()
	e.SetName("foo")
	suite.Equal(Name(&e), "foo")
}

func (suite *HelpersTestSuite) TestCName() {
	e := NewEntryBase()
	e.SetName("foo/bar/baz")
	suite.Equal("foo#bar#baz", CName(&e))

	e.SetSlashReplacer(':')
	suite.Equal("foo:bar:baz", CName(&e))
}

func (suite *HelpersTestSuite) TestID() {
	e := NewEntryBase()
	e.SetName("foo/bar")

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
		EntryBase: NewEntryBase(),
	}
	e.SetName("mockEntry")
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

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
