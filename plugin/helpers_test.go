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
	e := newHelpersTestsMockEntry("foo")
	suite.Equal(Name(e), "foo")
}

func (suite *HelpersTestSuite) TestCName() {
	e := newHelpersTestsMockEntry("foo/bar/baz")
	suite.Equal("foo#bar#baz", CName(e))

	e.SetSlashReplacer(':')
	suite.Equal("foo:bar:baz", CName(e))
}

func (suite *HelpersTestSuite) TestID() {
	e := newHelpersTestsMockEntry("foo/bar")

	e.SetTestID("")
	suite.Panics(
		func() { ID(e) },
		"plugin.ID: entry foo (cname foo#bar) has no ID",
	)

	e.SetTestID("/foo/bar")
	suite.Equal(ID(e), "/foo/bar")
}

type helpersTestsMockEntry struct {
	EntryBase
	mock.Mock
}

func (e *helpersTestsMockEntry) Schema() *EntrySchema {
	return nil
}

func newHelpersTestsMockEntry(name string) *helpersTestsMockEntry {
	e := &helpersTestsMockEntry{
		EntryBase: NewEntry(name),
	}
	e.SetTestID("id")
	e.DisableDefaultCaching()

	return e
}

func (suite *HelpersTestSuite) TestAttributes() {
	e := newHelpersTestsMockEntry("mockEntry")
	e.attr = EntryAttributes{}
	e.attr.SetCtime(time.Now())
	suite.Equal(e.attr, Attributes(e))
}

func (suite *HelpersTestSuite) TestPrefetched() {
	e := newHelpersTestsMockEntry("mockEntry")
	suite.False(IsPrefetched(e))
	e.Prefetched()
	suite.True(IsPrefetched(e))
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
