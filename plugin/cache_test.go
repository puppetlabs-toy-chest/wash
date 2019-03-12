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

func (suite *CacheTestSuite) SetupTest() {
	suite.cache = &cacheTestsMockCache{}
	SetTestCache(suite.cache)
}

func (suite *CacheTestSuite) TearDownTest() {
	UnsetTestCache()
}

func (suite *CacheTestSuite) TestDefaultCacheConfig() {
	config := newCacheConfig()
	assertOpTTL := func(op cachedOp, opName string, expectedTTL time.Duration) {
		actualTTL := config.getTTLOf(op)
		suite.Equal(
			expectedTTL,
			actualTTL,
			"expected the TTL of %v to be %v, but got %v instead",
			opName,
			expectedTTL,
			actualTTL,
		)
	}

	assertOpTTL(List, "List", 15*time.Second)
	assertOpTTL(Open, "Open", 15*time.Second)
	assertOpTTL(Metadata, "Metadata", 15*time.Second)
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
	mock.Mock
}

func (e *cacheTestsMockEntry) Name() string {
	return ""
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

type cacheTestsMockCacheableEntry struct {
	*cacheTestsMockEntry
	cacheConfig *CacheConfig
}

func newCacheTestsMockCacheableEntry() *cacheTestsMockCacheableEntry {
	return &cacheTestsMockCacheableEntry{
		&cacheTestsMockEntry{},
		newCacheConfig(),
	}
}

func (e *cacheTestsMockCacheableEntry) CacheConfig() *CacheConfig {
	return e.cacheConfig
}

type cachedOpFunc func(ctx context.Context, e Entry, id string) (interface{}, error)

func (suite *CacheTestSuite) testCachedOp(op cachedOp, opName string, mockValue interface{}, cachedOp cachedOpFunc) {
	ctx := context.Background()

	// Test that cachedOp panics if the cache == nil
	panicFunc := func() {
		entry := &cacheTestsMockEntry{}
		_, _ = cachedOp(ctx, entry, "id")
	}
	UnsetTestCache()
	suite.Panics(panicFunc)
	SetTestCache(suite.cache)

	// Test that cachedOp returns e.Op for an uncacheable entry
	entry := &cacheTestsMockEntry{}
	entry.On(opName, ctx).Return(mockValue, nil)
	v, err := cachedOp(ctx, entry, "id")
	if suite.NoError(err) {
		suite.Equal(mockValue, v)
	}
	suite.cache.AssertNotCalled(suite.T(), "GetOrUpdate")

	// Test that cachedOp does _not_ call cache#GetOrUpdate for a cacheable
	// entry that's turned off caching
	cacheableEntry := newCacheTestsMockCacheableEntry()
	cacheableEntry.On(opName, ctx).Return(mockValue, nil)
	cacheableEntry.CacheConfig().TurnOffCachingFor(op)
	v, err = cachedOp(ctx, entry, "id")
	if suite.NoError(err) {
		suite.Equal(mockValue, v)
	}
	suite.cache.AssertNotCalled(suite.T(), "GetOrUpdate")

	// Test that cachedOp does call cache#GetOrUpdate for a cacheable
	// entry, and that it passes-in the right arguments.
	opTTL := 5 * time.Second
	cacheableEntry.CacheConfig().SetTTLOf(op, opTTL)
	cacheableEntry.On(opName, ctx).Return(mockValue, nil)
	opKey := opName + "::" + "id"
	generateValueMatcher := func(generateValue func() (interface{}, error)) bool {
		// This matcher ensures that cachedOp is passing-in the right generator function to
		// GetOrUpdate
		res, err := generateValue()
		if suite.NoError(err) {
			return suite.Equal(mockValue, res)
		}

		return false
	}
	suite.cache.On("GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher)).Return(mockValue, nil).Once()
	v, err = cachedOp(ctx, cacheableEntry, "id")
	if suite.NoError(err) {
		suite.Equal(mockValue, v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher))
}

func (suite *CacheTestSuite) TestCachedList() {
	mockChildren := []Entry{&cacheTestsMockEntry{}}
	suite.testCachedOp(List, "List", mockChildren, func(ctx context.Context, e Entry, id string) (interface{}, error) {
		return CachedList(ctx, e.(Group), id)
	})
}

func (suite *CacheTestSuite) TestCachedOpen() {
	mockReader := strings.NewReader("foo")
	suite.testCachedOp(Open, "Open", mockReader, func(ctx context.Context, e Entry, id string) (interface{}, error) {
		return CachedOpen(ctx, e.(Readable), id)
	})
}

func (suite *CacheTestSuite) TestCachedMetadata() {
	mockMetadataMap := MetadataMap{"foo": "bar"}
	suite.testCachedOp(Metadata, "Metadata", mockMetadataMap, func(ctx context.Context, e Entry, id string) (interface{}, error) {
		return CachedMetadata(ctx, e.(Resource), id)
	})
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
