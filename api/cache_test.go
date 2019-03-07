package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CacheHandlerTestSuite struct {
	suite.Suite
	router *mux.Router
}

func (suite *CacheHandlerTestSuite) SetupSuite() {
	plugin.InitCache()
	suite.router = mux.NewRouter()
	suite.router.Handle("/cache/{path:.*}", cacheHandler)
}

func (suite *CacheHandlerTestSuite) TearDownSuite() {
	plugin.TeardownCache()
}

func (suite *CacheHandlerTestSuite) TestRejectsGet() {
	req := httptest.NewRequest(http.MethodGet, "http://example.com/cache/foo", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusNotFound, w.Code)
	var resp apitypes.ErrorObj
	if suite.Nil(json.Unmarshal(w.Body.Bytes(), &resp)) {
		suite.Equal("puppetlabs.wash/http-method-not-supported", resp.Kind)
		suite.Equal("The GET method is not supported for /cache/foo, supported methods are: DELETE", resp.Msg)
		suite.Equal(apitypes.ErrorFields{"method": "GET", "path": "/cache/foo", "supported": []interface{}{"DELETE"}}, resp.Fields)
	}
}

func (suite *CacheHandlerTestSuite) TestClearCache() {
	// Populate the cache with a mocked resource and plugin.Cached*
	var group mockedGroup
	group.On("List", mock.Anything).Return([]plugin.Entry{}, nil)

	if children, err := plugin.CachedList(context.Background(), &group, "/dir"); suite.Nil(err) {
		suite.Equal([]plugin.Entry{}, children)
	}

	// Test clearing a different cache
	req := httptest.NewRequest(http.MethodDelete, "http://example.com/cache/file", nil)
	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal("[]\n", w.Body.String())

	if children, err := plugin.CachedList(context.Background(), &group, "/dir"); suite.Nil(err) {
		suite.Equal([]plugin.Entry{}, children)
	}

	// Test clearing the cache
	req = httptest.NewRequest(http.MethodDelete, "http://example.com/cache/dir", nil)
	w = httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)
	suite.Equal(`["List::/dir"]`, strings.TrimSpace(w.Body.String()))

	if children, err := plugin.CachedList(context.Background(), &group, "/dir"); suite.Nil(err) {
		suite.Equal([]plugin.Entry{}, children)
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

func (g *mockedGroup) List(ctx context.Context) ([]plugin.Entry, error) {
	args := g.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}
