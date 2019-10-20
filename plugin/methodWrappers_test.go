package plugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type MethodWrappersTestSuite struct {
	suite.Suite
	cache *cacheTestsMockCache
}

func (suite *MethodWrappersTestSuite) SetupTest() {
	suite.cache = &cacheTestsMockCache{}
	SetTestCache(suite.cache)
}

func (suite *MethodWrappersTestSuite) TearDownTest() {
	UnsetTestCache()
	suite.cache = nil
}

func (suite *MethodWrappersTestSuite) TestName() {
	e := newMethodWrappersTestsMockEntry("foo")
	suite.Equal(Name(e), "foo")
}

func (suite *MethodWrappersTestSuite) TestCName() {
	e := newMethodWrappersTestsMockEntry("foo/bar/baz")
	suite.Equal("foo#bar#baz", CName(e))

	e.SetSlashReplacer(':')
	suite.Equal("foo:bar:baz", CName(e))
}

func (suite *MethodWrappersTestSuite) TestID() {
	e := newMethodWrappersTestsMockEntry("foo/bar")

	e.SetTestID("")
	suite.Panics(
		func() { ID(e) },
		"plugin.ID: entry foo (cname foo#bar) has no ID",
	)

	e.SetTestID("/foo/bar")
	suite.Equal(ID(e), "/foo/bar")
}

type methodWrappersTestsMockEntry struct {
	EntryBase
	mock.Mock
}

func (m *methodWrappersTestsMockEntry) Schema() *EntrySchema {
	return nil
}

func (m *methodWrappersTestsMockEntry) List(ctx context.Context) ([]Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (m *methodWrappersTestsMockEntry) Delete(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *methodWrappersTestsMockEntry) Open(ctx context.Context) (SizedReader, error) {
	args := m.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func newMethodWrappersTestsMockEntry(name string) *methodWrappersTestsMockEntry {
	e := &methodWrappersTestsMockEntry{
		EntryBase: NewEntry(name),
	}
	e.SetTestID("id")
	e.DisableDefaultCaching()

	return e
}

func (suite *MethodWrappersTestSuite) TestAttributes() {
	suite.cache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)

	e := newMethodWrappersTestsMockEntry("mockEntry")
	e.attr = EntryAttributes{}
	e.attr.SetCtime(time.Now())
	suite.Equal(e.attr, Attributes(e))
}

func (suite *MethodWrappersTestSuite) TestPrefetched() {
	e := newMethodWrappersTestsMockEntry("mockEntry")
	suite.False(IsPrefetched(e))
	e.Prefetched()
	suite.True(IsPrefetched(e))
}

func (suite *MethodWrappersTestSuite) TestDelete_ReturnsDeleteError() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("foo")

	expectedErr := fmt.Errorf("an error")
	e.On("Delete", ctx).Return(expectedErr)

	var entry Entry = e
	err := Delete(ctx, entry.(Deletable))
	suite.Equal(expectedErr, err)
}

func (suite *MethodWrappersTestSuite) TestDelete_DeletesEntry() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("foo")
	e.On("Delete", ctx).Return(nil)

	suite.cache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
	suite.cache.On("Delete", mock.Anything).Return([]string{})

	var entry Entry = e
	err := Delete(ctx, entry.(Deletable))
	if suite.NoError(err) {
		e.AssertExpectations(suite.T())
	}
}

func (suite *MethodWrappersTestSuite) TestDelete_ClearsEntryCache() {
	e := newMethodWrappersTestsMockEntry("foo")
	e.On("Delete", mock.Anything).Return(nil)

	suite.cache.On("Get", mock.Anything, mock.Anything).Return(nil, nil)
	suite.cache.On("Delete", opKeysRegex(e.id())).Return([]string{})

	var entry Entry = e
	err := Delete(context.Background(), entry.(Deletable))
	if suite.NoError(err) {
		suite.cache.AssertExpectations(suite.T())
	}
}

func (suite *MethodWrappersTestSuite) TestDelete_DeletesEntryFromParentsCachedEntryMap() {
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")
	e.On("Delete", mock.Anything).Return(nil)

	entryMap := newEntryMap()
	entryMap.mp["bar"] = e
	suite.cache.On("Get", "List", "/foo").Return(entryMap, nil)
	suite.cache.On("Delete", mock.Anything).Return([]string{})

	var entry Entry = e
	err := Delete(context.Background(), entry.(Deletable))
	if suite.NoError(err) {
		suite.cache.AssertCalled(suite.T(), "Get", "List", "/foo")
		suite.NotContains(entryMap.mp, "bar")
	}
}

func TestMethodWrappers(t *testing.T) {
	suite.Run(t, new(MethodWrappersTestSuite))
}
