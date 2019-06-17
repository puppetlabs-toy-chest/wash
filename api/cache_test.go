package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
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

func (m *mockCache) GetOrUpdate(cat, key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error) {
	key = cat + "::" + key
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
	parent := newMockedParent()
	parent.SetTestID("/dir")
	parent.On("List", mock.Anything).Return([]plugin.Entry{}, nil)

	reqCtx := context.WithValue(context.Background(), mountpointKey, "/")

	expectedChildren := make(map[string]plugin.Entry)
	if children, err := plugin.CachedList(reqCtx, parent); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	// Test clearing a different cache
	req := httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=/file", nil).WithContext(reqCtx)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal("[]\n", w.Body.String())

	if children, err := plugin.CachedList(context.Background(), parent); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	// Test clearing the cache
	req = httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=/dir", nil).WithContext(reqCtx)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal(`["List::/dir"]`, strings.TrimSpace(w.Body.String()))

	if children, err := plugin.CachedList(context.Background(), parent); suite.Nil(err) {
		suite.Equal(expectedChildren, children)
	}

	parent.AssertNumberOfCalls(suite.T(), "List", 2)
}

func (suite *CacheHandlerTestSuite) TestClearCacheErrors() {
	reqCtx := context.WithValue(context.Background(), mountpointKey, "/mnt")

	// Test clearing cache by a relative path
	req := httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=mnt/file", nil).WithContext(reqCtx)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusBadRequest, w.Code)
	var errResp apitypes.ErrorObj
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &errResp))
	suite.Equal(apitypes.RelativePath, errResp.Kind)

	// Test clearing cache outside the mountpoint
	req = httptest.NewRequest(http.MethodDelete, "http://example.com/cache?path=/a/file", nil).WithContext(reqCtx)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusBadRequest, w.Code)
	suite.NoError(json.Unmarshal(w.Body.Bytes(), &errResp))
	suite.Equal(apitypes.NonWashPath, errResp.Kind)
}

func TestCacheHandler(t *testing.T) {
	suite.Run(t, new(CacheHandlerTestSuite))
}

type mockedParent struct {
	plugin.EntryBase
	mock.Mock
}

func newMockedParent() *mockedParent {
	p := &mockedParent{
		EntryBase: plugin.NewEntryBase(),
	}
	p.SetName("mockParent")
	return p
}

func (p *mockedParent) List(ctx context.Context) ([]plugin.Entry, error) {
	args := p.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}

func (p *mockedParent) ChildSchemas() []*plugin.EntrySchema {
	return nil
}
