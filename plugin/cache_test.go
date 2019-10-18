package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type cacheTestsMockCache struct {
	mock.Mock
}

func (m *cacheTestsMockCache) Get(cat, key string) (interface{}, error) {
	args := m.Called(cat, key)
	return args.Get(0), args.Error(1)
}

func (m *cacheTestsMockCache) GetOrUpdate(cat, key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error) {
	args := m.Called(cat, key, ttl, resetTTLOnHit, generateValue)
	return args.Get(0), args.Error(1)
}

func (m *cacheTestsMockCache) Flush() {
	// Don't need anything for Flush, so leave it alone for now
}

func (m *cacheTestsMockCache) Delete(matcher *regexp.Regexp) []string {
	args := m.Called(matcher)
	return args.Get(0).([]string)
}

type CacheTestSuite struct {
	suite.Suite
	cache *cacheTestsMockCache
}

type generateValueMatcherFunc = func(func() (interface{}, error)) bool

func (suite *CacheTestSuite) makeGenerateValueMatcher(expectedValue interface{}) generateValueMatcherFunc {
	return func(generateValue func() (interface{}, error)) bool {
		res, err := generateValue()
		if suite.NoError(err) {
			return suite.Equal(expectedValue, res)
		}

		return false
	}
}

func (suite *CacheTestSuite) SetupTest() {
	suite.cache = &cacheTestsMockCache{}
	SetTestCache(suite.cache)
}

func (suite *CacheTestSuite) TearDownTest() {
	UnsetTestCache()
}

func (suite *CacheTestSuite) opKeysRegex(path string) *regexp.Regexp {
	rx, err := opKeysRegex(path)
	if err != nil {
		suite.FailNow(
			fmt.Sprintf("opKeysRegex unexpectedly errored with %v", err),
		)
	}

	return rx
}

func (suite *CacheTestSuite) TestOpNameRegex() {
	suite.Regexp(opNameRegex, "a")
	suite.Regexp(opNameRegex, "A")
	suite.Regexp(opNameRegex, "op")
	suite.Regexp(opNameRegex, "Op")
	suite.Regexp(opNameRegex, "List")
	suite.Regexp(opNameRegex, "Open")
	suite.Regexp(opNameRegex, "Metadata")

	suite.NotRegexp(opNameRegex, "")
	suite.NotRegexp(opNameRegex, " op")
	suite.NotRegexp(opNameRegex, "123")
	suite.NotRegexp(opNameRegex, "abc  ")
}

func (suite *CacheTestSuite) TestOpKeysRegex() {
	rx := suite.opKeysRegex("/a")

	// Test that it matches children
	suite.Regexp(rx, "Test::/a/b")
	suite.Regexp(rx, "Test::/a/b/c")
	suite.Regexp(rx, "Test::/a/bcd/ef/g")
	suite.Regexp(rx, "Test::/a/a space")

	// Test that it does not match other entries
	suite.NotRegexp(rx, "Test::/")
	suite.NotRegexp(rx, "Test::/ab")
	suite.NotRegexp(rx, "Test::/bc/d")

	// Test that it matches root, and children of root
	rx = suite.opKeysRegex("/")
	suite.Regexp(rx, "Test::/")
	suite.Regexp(rx, "Test::/a")
	suite.Regexp(rx, "Test::/a/b")

}

func (suite *CacheTestSuite) TestClearCache() {
	path := "/a"
	rx := suite.opKeysRegex(path)

	suite.cache.On("Delete", rx).Return([]string{"/a"})
	deleted, err := ClearCacheFor(path)
	if !suite.NoError(err) {
		suite.Equal([]string{"/a"}, deleted)
	}
}

type cacheTestsMockEntry struct {
	EntryBase
	mock.Mock
}

func newCacheTestsMockEntry(name string) *cacheTestsMockEntry {
	e := &cacheTestsMockEntry{
		EntryBase: NewEntry(name),
	}
	return e
}

func (e *cacheTestsMockEntry) List(ctx context.Context) ([]Entry, error) {
	args := e.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (e *cacheTestsMockEntry) ChildSchemas() []*EntrySchema {
	return nil
}

func (e *cacheTestsMockEntry) Schema() *EntrySchema {
	return nil
}

func (e *cacheTestsMockEntry) Open(ctx context.Context) (SizedReader, error) {
	args := e.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func (e *cacheTestsMockEntry) Metadata(ctx context.Context) (JSONObject, error) {
	args := e.Called(ctx)
	return args.Get(0).(JSONObject), args.Error(1)
}

type cachedDefaultOpFunc func(ctx context.Context, e Entry) (interface{}, error)

func (suite *CacheTestSuite) TestCachedOp() {
	makePanicFunc := func(opName string, ttl time.Duration) func() {
		return func() {
			entry := newCacheTestsMockEntry("mock")
			_, _ = CachedOp(context.Background(), "List", entry, ttl, func() (interface{}, error) { return nil, nil })
		}
	}

	// Test that CachedOp panics if opName does not match opNameRegex
	suite.Panics(makePanicFunc("123", 15), fmt.Sprintf("The opName 123 does not match %v", opNameRegex.String()))

	// Test that CachedOp panics if an opName == an action op's name
	suite.Panics(makePanicFunc("List", 15), "The opName List conflicts with CachedList")

	// Test that CachedOp panics if a negative TTL's passed-in
	suite.Panics(makePanicFunc("Op", -15), "plugin.CachedOp: received a negative TTL")

	// Test that CachedOp panics if the cache == nil
	UnsetTestCache()
	suite.Panics(makePanicFunc("Op", 15))
	SetTestCache(suite.cache)

	// Test that CachedOp panics if entry.id() == ""
	suite.Panics(makePanicFunc("Op", 15), "entry.id() returned an empty ID")

	// Test that CachedOp calls cache#GetOrUpdate with the right parameters
	entry := newCacheTestsMockEntry("mock")
	entry.SetTestID("id")
	opName := "Op"
	opTTL := 5 * time.Second
	op := func() (interface{}, error) { return "result", nil }
	generateValueMatcher := suite.makeGenerateValueMatcher("result")
	suite.cache.On("GetOrUpdate", opName, entry.id(), opTTL, false, mock.MatchedBy(generateValueMatcher)).Return("result", nil).Once()
	v, err := CachedOp(context.Background(), opName, entry, opTTL, op)
	if suite.NoError(err) {
		suite.Equal("result", v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opName, entry.id(), opTTL, false, mock.MatchedBy(generateValueMatcher))
}

func (suite *CacheTestSuite) TestDuplicateCNameErr() {
	err := DuplicateCNameErr{
		ParentID:                 "/my_plugin/foo",
		FirstChildName:           "foo/bar/",
		FirstChildSlashReplacer:  '#',
		SecondChildName:          "foo#bar/",
		SecondChildSlashReplacer: '#',
		CName:                    "foo#bar#",
	}

	suite.Regexp("listing /my_plugin/foo", err.Error())
	suite.Regexp("foo/bar/.*foo#bar/.*foo#bar#", err.Error())
	suite.Regexp("my_plugin plugin", err.Error())
}

func (suite *CacheTestSuite) testCachedDefaultOp(
	op defaultOpCode,
	opName string,
	opValue interface{},
	mungedOpValue interface{},
	cachedDefaultOp cachedDefaultOpFunc,
) {
	ctx := context.Background()

	// Test that cachedDefaultOp panics if the cache == nil
	panicFunc := func() {
		entry := newCacheTestsMockEntry("mock")
		_, _ = cachedDefaultOp(ctx, entry)
	}
	UnsetTestCache()
	suite.Panics(panicFunc)
	SetTestCache(suite.cache)

	entry := newCacheTestsMockEntry("mock")

	// Test that cachedDefaultOp does _not_ call cache#GetOrUpdate for an
	// entry that's turned off caching
	entry.On(opName, mock.Anything).Return(opValue, nil)
	entry.DisableCachingFor(op)
	v, err := cachedDefaultOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mungedOpValue, v)
	}
	suite.cache.AssertNotCalled(suite.T(), "GetOrUpdate")

	// Test that cachedDefaultOp panics if entry.id() == "" if not passed
	// a suitable context.
	suite.Panics(panicFunc, "entry.id() returned an empty ID")
	entry.SetTestID("id")

	// Test that cachedDefaultOp does call cache#GetOrUpdate for an
	// entry that's enabled caching, and that it passes-in the
	// right arguments.
	opTTL := 5 * time.Second
	entry.SetTTLOf(op, opTTL)
	entry.On(opName, mock.Anything).Return(opValue, nil)
	generateValueMatcher := suite.makeGenerateValueMatcher(mungedOpValue)
	suite.cache.On("GetOrUpdate", opName, entry.id(), opTTL, false, mock.MatchedBy(generateValueMatcher)).Return(mungedOpValue, nil).Once()
	v, err = cachedDefaultOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mungedOpValue, v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opName, entry.id(), opTTL, false, mock.MatchedBy(generateValueMatcher))
}

func toMap(children []Entry) map[string]Entry {
	mp := make(map[string]Entry)
	for _, child := range children {
		mp[CName(child)] = child
	}

	return mp
}

func (suite *CacheTestSuite) TestCachedListDefaultOp() {
	mockChildren := []Entry{newCacheTestsMockEntry("mockChild")}
	mungedOpValue := toMap(mockChildren)
	suite.testCachedDefaultOp(ListOp, "List", mockChildren, mungedOpValue, func(ctx context.Context, e Entry) (interface{}, error) {
		return cachedList(ctx, e.(Parent))
	})
}

func (suite *CacheTestSuite) TestCachedListCNameErrors() {
	ctx := context.Background()
	entry := newCacheTestsMockEntry("foo")
	entry.DisableDefaultCaching()
	entry.SetTestID("/my_plugin/foo")

	// Test that CachedList returns an error if two children have the same
	// cname
	child1 := newCacheTestsMockEntry("foo/bar/")
	child2 := newCacheTestsMockEntry("foo#bar/")
	child3 := newCacheTestsMockEntry("baz")
	mockChildren := []Entry{child1, child2, child3}
	entry.On("List", mock.Anything).Return(mockChildren, nil).Once()
	_, err := cachedList(ctx, entry)
	if suite.Error(err) {
		expectedErr := DuplicateCNameErr{
			ParentID:                 "/my_plugin/foo",
			FirstChildName:           "foo/bar/",
			FirstChildSlashReplacer:  '#',
			SecondChildName:          "foo#bar/",
			SecondChildSlashReplacer: '#',
			CName:                    "foo#bar#",
		}

		suite.Equal(expectedErr, err)
	}
}

func (suite *CacheTestSuite) TestCachedListSetEntryID() {
	// Test that CachedList sets the children's cache IDs
	// to <parent_cache_id>/<child_cname>
	ctx := context.Background()
	child1 := newCacheTestsMockEntry("foo/child1")
	child2 := newCacheTestsMockEntry("child2")
	mockChildren := []Entry{child1, child2}

	// When the parent's the root
	entry := newCacheTestsMockEntry("/")
	entry.SetTestID("/")
	entry.DisableDefaultCaching()
	entry.On("List", mock.Anything).Return(mockChildren, nil).Once()
	children, err := cachedList(ctx, entry)
	if suite.NoError(err) {
		if suite.Equal(toMap(mockChildren), children) {
			suite.Equal("/foo#child1", children["foo#child1"].id())
			suite.Equal("/child2", children["child2"].id())
		}
	}

	// When the parent's some other entry
	entry = newCacheTestsMockEntry("parent")
	entry.SetTestID("/parent")
	entry.DisableDefaultCaching()
	entry.On("List", mock.Anything).Return(mockChildren, nil).Once()
	children, err = cachedList(ctx, entry)
	if suite.NoError(err) {
		if suite.Equal(toMap(mockChildren), children) {
			suite.Equal("/parent/foo#child1", children["foo#child1"].id())
			suite.Equal("/parent/child2", children["child2"].id())
		}
	}
}

func (suite *CacheTestSuite) TestCachedOpen() {
	mockReader := strings.NewReader("foo")
	suite.testCachedDefaultOp(OpenOp, "Open", mockReader, mockReader, func(ctx context.Context, e Entry) (interface{}, error) {
		return cachedOpen(ctx, e.(Readable))
	})
}

func (suite *CacheTestSuite) TestCachedMetadata() {
	mockJSONObject := JSONObject{"foo": "bar"}
	suite.testCachedDefaultOp(MetadataOp, "Metadata", mockJSONObject, mockJSONObject, func(ctx context.Context, e Entry) (interface{}, error) {
		return CachedMetadata(ctx, e)
	})
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
