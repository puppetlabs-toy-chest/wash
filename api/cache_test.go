package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// TODO: Now that the plugin cache can be mocked, come back later and
// simplify these tests.

type mockCache struct {
	mock.Mock
	items map[string]interface{}
}

func newMockCache() *mockCache {
	return &mockCache{items: make(map[string]interface{})}
}

func (m *mockCache) GetOrUpdate(key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error) {
	if v, ok := m.items[key]; ok {
		return v, nil
	}

	val, err := generateValue()
	if err != nil {
		return nil, err
	}

	m.items[key] = val
	return val, nil
}

func (m *mockCache) Flush() {
	m.items = make(map[string]interface{})
}

func (m *mockCache) Delete(matcher *regexp.Regexp) []string {
	deleted := make([]string, 0, len(m.items))
	for k := range m.items {
		if matcher.MatchString(k) {
			delete(m.items, k)
			deleted = append(deleted, k)
		}
	}

	return deleted
}

type CacheHandlerTestSuite struct {
	suite.Suite
	router *mux.Router
}

func (suite *CacheHandlerTestSuite) SetupSuite() {
	plugin.SetTestCache(newMockCache())
	suite.router = mux.NewRouter()
	suite.router.Handle("/cache", cacheHandler).Methods(http.MethodDelete)
}

func (suite *CacheHandlerTestSuite) TearDownSuite() {
	plugin.UnsetTestCache()
}

func (suite *CacheHandlerTestSuite) TestRejectsGet() {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/cache", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusMethodNotAllowed, w.Code)
}

func (suite *CacheHandlerTestSuite) TestClearCache() {
	// Populate the cache with a mocked resource and plugin.Cached*
	group := newMockedGroup()
	group.On("List", mock.Anything).Return([]plugin.Entry{}, nil)

	expectedChildren := make(map[string]plugin.Entry)
	if children, err := plugin.CachedList(context.Background(), group); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	// Test clearing a different cache
	req := httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=file", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal("[]\n", w.Body.String())

	if children, err := plugin.CachedList(context.Background(), group); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	// Test clearing the cache
	req = httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=%2FmockGroup", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal(`["List::/mockGroup"]`, strings.TrimSpace(w.Body.String()))

	if children, err := plugin.CachedList(context.Background(), group); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	group.AssertNumberOfCalls(suite.T(), "List", 2)
}

func TestCacheHandler(t *testing.T) {
	suite.Run(t, new(CacheHandlerTestSuite))
}

type mockedGroup struct {
	plugin.EntryBase
	mock.Mock
}

func newMockedGroup() *mockedGroup {
	g := &mockedGroup{
		EntryBase: plugin.NewRootEntry("mockGroup"),
	}

	return g
}

func (g *mockedGroup) List(ctx context.Context) ([]plugin.Entry, error) {
	args := g.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}
