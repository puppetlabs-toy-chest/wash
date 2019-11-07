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
	args := m.Called()
	return args.Get(0).(*EntrySchema)
}

func (m *methodWrappersTestsMockEntry) List(ctx context.Context) ([]Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (m *methodWrappersTestsMockEntry) Delete(ctx context.Context) (bool, error) {
	args := m.Called(ctx)
	return args.Get(0).(bool), args.Error(1)
}

func (m *methodWrappersTestsMockEntry) Open(ctx context.Context) (SizedReader, error) {
	args := m.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func (m *methodWrappersTestsMockEntry) Signal(ctx context.Context, signal string) error {
	args := m.Called(ctx, signal)
	return args.Error(0)
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

func (suite *MethodWrappersTestSuite) TestSignal_ReturnsSignalError() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("foo")

	expectedErr := fmt.Errorf("an error")

	e.On("Schema").Return((*EntrySchema)(nil))
	e.On("Signal", ctx, "start").Return(expectedErr)

	err := Signal(ctx, e, "start")
	suite.Equal(expectedErr, err)
}

func (suite *MethodWrappersTestSuite) TestSignal_SendsSignalAndUpdatesCache() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")

	e.On("Schema").Return((*EntrySchema)(nil))
	e.On("Signal", ctx, "start").Return(nil)

	suite.cache.On("Delete", allOpKeysIncludingChildrenRegex(e.id())).Return([]string{})
	suite.cache.On("Get", "List", "/foo").Return(newEntryMap(), nil)
	suite.cache.On("Delete", opKeyRegex("List", "/foo")).Return([]string{})

	// Also test case-insensitivity here
	err := Signal(ctx, e, "START")
	if suite.NoError(err) {
		e.AssertExpectations(suite.T())
		suite.cache.AssertExpectations(suite.T())
	}
}

func (suite *MethodWrappersTestSuite) TestSignal_SchemaKnown_ReturnsInvalidInputErrForInvalidSignal() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("foo")

	schema := &EntrySchema{
		entrySchema: entrySchema{
			Signals: map[string]string{
				"start": "Starts the entry",
				"stop":  "Stops the entry",
			},
		},
	}
	e.On("Schema").Return(schema)
	e.On("Signal", ctx, "start").Return(nil)

	err := Signal(ctx, e, "invalid_signal")
	suite.True(IsInvalidInputErr(err))
	suite.Regexp("invalid.*signal.*invalid_signal.*start.*stop", err)
}

func (suite *MethodWrappersTestSuite) TestDelete_ReturnsDeleteError() {
	ctx := context.Background()
	e := newMethodWrappersTestsMockEntry("foo")

	expectedErr := fmt.Errorf("an error")
	e.On("Delete", ctx).Return(false, expectedErr)

	_, err := Delete(ctx, e)
	suite.Equal(expectedErr, err)
}

func (suite *MethodWrappersTestSuite) TestDelete_EntryDeletionInProgress_UpdatesCache() {
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")
	e.On("Delete", mock.Anything).Return(false, nil)

	suite.cache.On("Delete", allOpKeysIncludingChildrenRegex(e.id())).Return([]string{})
	suite.cache.On("Get", "List", "/foo").Return(newEntryMap(), nil)
	suite.cache.On("Delete", opKeyRegex("List", "/foo")).Return([]string{})

	deleted, err := Delete(context.Background(), e)
	if suite.NoError(err) {
		suite.False(deleted)
		e.AssertExpectations(suite.T())
		suite.cache.AssertExpectations(suite.T())
	}
}

func (suite *MethodWrappersTestSuite) TestDelete_EntryDeletionInProgress_NoCachedListResult_IgnoresParentCache() {
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")
	e.On("Delete", mock.Anything).Return(false, nil)

	suite.cache.On("Delete", mock.Anything).Return([]string{})
	suite.cache.On("Get", "List", "/foo").Return(nil, nil)

	deleted, err := Delete(context.Background(), e)
	if suite.NoError(err) {
		suite.False(deleted)
		suite.cache.AssertNotCalled(suite.T(), "Delete", opKeyRegex("List", "/foo"))
	}
}

func (suite *MethodWrappersTestSuite) TestDelete_DeletedEntry_UpdatesCache() {
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")
	e.On("Delete", mock.Anything).Return(true, nil)

	entryMap := newEntryMap()
	entryMap.mp["bar"] = e
	suite.cache.On("Delete", allOpKeysIncludingChildrenRegex(e.id())).Return([]string{})
	suite.cache.On("Get", "List", "/foo").Return(entryMap, nil)

	deleted, err := Delete(context.Background(), e)
	if suite.NoError(err) {
		suite.True(deleted)
		e.AssertExpectations(suite.T())
		suite.cache.AssertExpectations(suite.T())
		suite.NotContains(entryMap.mp, "bar")
	}
}

func (suite *MethodWrappersTestSuite) TestDelete_DeletedEntry_NoCachedListResult_IgnoresParentCache() {
	e := newMethodWrappersTestsMockEntry("bar")
	e.SetTestID("/foo/bar")
	e.On("Delete", mock.Anything).Return(true, nil)

	suite.cache.On("Delete", mock.Anything).Return([]string{})
	suite.cache.On("Get", "List", "/foo").Return(nil, nil)

	deleted, err := Delete(context.Background(), e)
	if suite.NoError(err) {
		suite.True(deleted)
		// If Delete did not ignore the parent cache, then there'd be a nil pointer panic
		// because Delete calls EntryMap#Delete. Thus if we get to this point, that means
		// a panic did not occur so the test passed.
	}
}

func TestMethodWrappers(t *testing.T) {
	suite.Run(t, new(MethodWrappersTestSuite))
}
