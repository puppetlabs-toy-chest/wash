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

func newCacheTestsRootEntry(name string) *cacheTestsMockEntry {
	return &cacheTestsMockEntry{EntryBase: NewRootEntry(name)}
}

func newCacheTestsMockEntry(name string) *cacheTestsMockEntry {
	e := new(cacheTestsMockEntry)
	e.EntryBase = e.NewEntry(name)
	return e
}

func (e *cacheTestsMockEntry) List(ctx context.Context) ([]Entry, error) {
	args := e.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (e *cacheTestsMockEntry) Open(ctx context.Context) (SizedReader, error) {
	args := e.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func (e *cacheTestsMockEntry) Metadata(ctx context.Context) (EntryMetadata, error) {
	args := e.Called(ctx)
	return args.Get(0).(EntryMetadata), args.Error(1)
}

type cachedDefaultOpFunc func(ctx context.Context, e Entry) (interface{}, error)

func (suite *CacheTestSuite) TestCachedOp() {
	makePanicFunc := func(opName string, ttl time.Duration) func() {
		return func() {
			entry := newCacheTestsMockEntry("mock")
			_, _ = CachedOp("List", entry, ttl, func() (interface{}, error) { return nil, nil })
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
	entry.washID = "id"
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

func (suite *CacheTestSuite) TestDuplicateCNameErr() {
	err := DuplicateCNameErr{
		ParentPath:                      "/my_plugin/foo",
		FirstChildName:                  "foo/bar/",
		FirstChildSlashReplacementChar:  '#',
		SecondChildName:                 "foo#bar/",
		SecondChildSlashReplacementChar: '#',
		CName:                           "foo#bar#",
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
	entry.washID = "id"
	entry.On(opName, ctx).Return(opValue, nil)
	entry.DisableCachingFor(op)
	v, err := cachedDefaultOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mungedOpValue, v)
	}
	suite.cache.AssertNotCalled(suite.T(), "GetOrUpdate")

	// Test that cachedDefaultOp does call cache#GetOrUpdate for an
	// entry that's enabled caching, and that it passes-in the
	// right arguments.
	opTTL := 5 * time.Second
	entry.SetTTLOf(op, opTTL)
	entry.On(opName, ctx).Return(opValue, nil)
	opKey := opName + "::" + "id"
	generateValueMatcher := suite.makeGenerateValueMatcher(mungedOpValue)
	suite.cache.On("GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher)).Return(mungedOpValue, nil).Once()
	v, err = cachedDefaultOp(ctx, entry)
	if suite.NoError(err) {
		suite.Equal(mungedOpValue, v)
	}
	suite.cache.AssertCalled(suite.T(), "GetOrUpdate", opKey, opTTL, false, mock.MatchedBy(generateValueMatcher))
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
		return CachedList(ctx, e.(Group))
	})
}

func (suite *CacheTestSuite) TestCachedListCNameErrors() {
	ctx := context.Background()
	entry := newCacheTestsMockEntry("foo")
	entry.DisableDefaultCaching()
	entry.washID = "/my_plugin/foo"

	// Test that CachedList returns an error if two children have the same
	// cname
	child1 := newCacheTestsMockEntry("foo/bar/")
	child2 := newCacheTestsMockEntry("foo#bar/")
	child3 := newCacheTestsMockEntry("baz")
	mockChildren := []Entry{child1, child2, child3}
	entry.On("List", ctx).Return(mockChildren, nil).Once()
	_, err := CachedList(ctx, entry)
	if suite.Error(err) {
		expectedErr := DuplicateCNameErr{
			ParentPath:                      "/my_plugin/foo",
			FirstChildName:                  "foo/bar/",
			FirstChildSlashReplacementChar:  '#',
			SecondChildName:                 "foo#bar/",
			SecondChildSlashReplacementChar: '#',
			CName:                           "foo#bar#",
		}

		suite.Equal(expectedErr, err)
	}
}

func (suite *CacheTestSuite) TestCachedListEntryID() {
	// Test that items returned from CachedList include cache IDs
	// to <parent_cache_id>/<child_cname>
	ctx := context.Background()

	// When the parent's the root
	entry := newCacheTestsRootEntry("/")
	entry.DisableDefaultCaching()

	child1 := entry.NewEntry("foo/child1")
	child2 := entry.NewEntry("child2")
	mockChildren := []Entry{&child1, &child2}

	entry.On("List", ctx).Return(mockChildren, nil).Once()
	children, err := CachedList(ctx, entry)
	if suite.NoError(err) {
		if suite.Equal(toMap(mockChildren), children) {
			suite.Equal("/foo#child1", children["foo#child1"].id())
			suite.Equal("/child2", children["child2"].id())
		}
	}

	// When the parent's some other entry
	entry = newCacheTestsRootEntry("parent")
	entry.DisableDefaultCaching()

	child1 = entry.NewEntry("foo/child1")
	child2 = entry.NewEntry("child2")
	mockChildren = []Entry{&child1, &child2}

	entry.On("List", ctx).Return(mockChildren, nil).Once()
	children, err = CachedList(ctx, entry)
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
		return CachedOpen(ctx, e.(Readable))
	})
}

func (suite *CacheTestSuite) TestCachedMetadata() {
	mockEntryMetadata := EntryMetadata{"foo": "bar"}
	suite.testCachedDefaultOp(MetadataOp, "Metadata", mockEntryMetadata, mockEntryMetadata, func(ctx context.Context, e Entry) (interface{}, error) {
		return CachedMetadata(ctx, e)
	})
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
