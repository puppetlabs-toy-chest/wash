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

func (m *cacheTestsMockCache) GetOrUpdate(key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error) {
	args := m.Called(key, ttl, resetTTLOnHit, generateValue)
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

func (suite *CacheTestSuite) TestOpKeysRegex() {
	rx := suite.opKeysRegex("/a")

	// Test that it matches all of the op keys
	suite.Regexp(rx, "List::/a")
	suite.Regexp(rx, "Open::/a")
	suite.Regexp(rx, "Metadata::/a")

	// Test that it matches children
	suite.Regexp(rx, "List::/a/b")
	suite.Regexp(rx, "List::/a/b/c")
	suite.Regexp(rx, "List::/a/bcd/ef/g")
	suite.Regexp(rx, "List::/a/a space")

	// Test that it does not match other entries
	suite.NotRegexp(rx, "List::/")
	suite.NotRegexp(rx, "List::/ab")
	suite.NotRegexp(rx, "List::/bc/d")

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
	return &cacheTestsMockEntry{
		EntryBase: newEntryBase(name),
	}
}

func (e *cacheTestsMockEntry) List(ctx context.Context) ([]Entry, error) {
	args := e.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (e *cacheTestsMockEntry) Open(ctx context.Context) (SizedReader, error) {
	args := e.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func (e *cacheTestsMockEntry) Metadata(ctx context.Context) (MetadataMap, error) {
	args := e.Called(ctx)
	return args.Get(0).(MetadataMap), args.Error(1)
}

type cachedActionOpFunc func(ctx context.Context, e Entry) (interface{}, error)

func (suite *CacheTestSuite) TestCachedOp() {
	makePanicFunc := func(opName string, ttl time.Duration) func() {
		return func() {
			entry := newCacheTestsMockEntry("mock")
			_, _ = CachedOp("List", entry, ttl, func() (interface{}, error) { return nil, nil })
		}
	}

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
	opKey := "Op::id"
	generateValueMatcher := suite.makeGenerateValueMatcher("result")
	suite.cache.On("GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher)).Return("result", nil).Once()
	v, err := CachedOp(opName, entry, opTTL, op)
	if suite.NoError(err) {
		suite.Equal("result", v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher))
}

func (suite *CacheTestSuite) testCachedActionOp(op actionOpCode, opName string, mockValue interface{}, cachedActionOp cachedActionOpFunc) {
	ctx := context.Background()

	// Test that cachedActionOp panics if the cache == nil
	panicFunc := func() {
		entry := newCacheTestsMockEntry("mock")
		_, _ = cachedActionOp(ctx, entry)
	}
	UnsetTestCache()
	suite.Panics(panicFunc)
	SetTestCache(suite.cache)

	// Test that cachedActionOp panics if entry.id() == ""
	suite.Panics(panicFunc, "entry.id() returned an empty ID")

	// Test that cachedActionOp does _not_ call cache#GetOrUpdate for an
	// entry that's turned off caching
	entry := newCacheTestsMockEntry("mock")
	entry.SetTestID("id")
	entry.On(opName, ctx).Return(mockValue, nil)
	entry.DisableCachingFor(op)
	v, err := cachedActionOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mockValue, v)
	}
	suite.cache.AssertNotCalled(suite.T(), "GetOrUpdate")

	// Test that cachedActionOp does call cache#GetOrUpdate for an
	// entry that's enabled caching, and that it passes-in the
	// right arguments.
	opTTL := 5 * time.Second
	entry.SetTTLOf(op, opTTL)
	entry.On(opName, ctx).Return(mockValue, nil)
	opKey := opName + "::" + "id"
	generateValueMatcher := suite.makeGenerateValueMatcher(mockValue)
	suite.cache.On("GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher)).Return(mockValue, nil).Once()
	v, err = cachedActionOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mockValue, v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher))
}

func (suite *CacheTestSuite) TestCachedList() {
	mockChildren := []Entry{newCacheTestsMockEntry("mockChild")}
	suite.testCachedActionOp(List, "List", mockChildren, func(ctx context.Context, e Entry) (interface{}, error) {
		return CachedList(ctx, e.(Group))
	})

	// Test that CachedList sets the children's cache IDs
	// to <parent_cache_id>/<child_name>
	ctx := context.Background()
	mockChildren = []Entry{
		newCacheTestsMockEntry("child1"),
		newCacheTestsMockEntry("child2"),
	}

	// When the parent's the root
	entry := newCacheTestsMockEntry("/")
	entry.SetTestID("/")
	entry.DisableDefaultCaching()
	entry.On("List", ctx).Return(mockChildren, nil).Once()
	children, err := CachedList(ctx, entry)
	if suite.NoError(err) {
		if suite.Equal(mockChildren, children) {
			suite.Equal("/child1", children[0].id())
			suite.Equal("/child2", children[1].id())
		}
	}

	// When the parent's some other entry
	entry = newCacheTestsMockEntry("parent")
	entry.SetTestID("/parent")
	entry.DisableDefaultCaching()
	entry.On("List", ctx).Return(mockChildren, nil).Once()
	children, err = CachedList(ctx, entry)
	if suite.NoError(err) {
		if suite.Equal(mockChildren, children) {
			suite.Equal("/parent/child1", children[0].id())
			suite.Equal("/parent/child2", children[1].id())
		}
	}
}

func (suite *CacheTestSuite) TestCachedOpen() {
	mockReader := strings.NewReader("foo")
	suite.testCachedActionOp(Open, "Open", mockReader, func(ctx context.Context, e Entry) (interface{}, error) {
		return CachedOpen(ctx, e.(Readable))
	})
}

func (suite *CacheTestSuite) TestCachedMetadata() {
	mockMetadataMap := MetadataMap{"foo": "bar"}
	suite.testCachedActionOp(Metadata, "Metadata", mockMetadataMap, func(ctx context.Context, e Entry) (interface{}, error) {
		return CachedMetadata(ctx, e.(Resource))
	})
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
