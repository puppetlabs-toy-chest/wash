package plugin

import (
	"sort"
	"testing"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type updater struct {
	mock.Mock
}

func (u *updater) Update() (interface{}, error) {
	args := u.Called()
	return args.Get(0), args.Error(1)
}

func cacheInit() (interface{}, error) { return struct{}{}, nil }

type CacheTestSuite struct {
	suite.Suite
}

func (suite *CacheTestSuite) SetupSuite() {
	SetTestCache(datastore.NewMemCache())
}

func (suite *CacheTestSuite) TearDownSuite() {
	UnsetTestCache()
}

func (suite *CacheTestSuite) getOrUpdate(key string, init func() (interface{}, error)) {
	_, err := cache.GetOrUpdate(key, 30*time.Second, false, init)
	suite.Nil(err)
}

func (suite *CacheTestSuite) TestClearCache() {
	suite.getOrUpdate("Test::/", cacheInit)
	deleted, err := ClearCacheFor("/")
	if suite.Nil(err) {
		suite.Equal([]string{"Test::/"}, deleted)
	}

	var m updater
	m.On("Update").Return("hello", nil)
	suite.getOrUpdate("Test::/", m.Update)
	m.AssertExpectations(suite.T())
}

func (suite *CacheTestSuite) TestClearMultiple() {
	suite.getOrUpdate("Test::/", cacheInit)
	suite.getOrUpdate("Test::/a", cacheInit)
	suite.getOrUpdate("Test::/a/b", cacheInit)
	suite.getOrUpdate("Test::/ab", cacheInit)

	deleted, err := ClearCacheFor("a")
	if suite.Nil(err) {
		sort.Strings(deleted)
		suite.Equal([]string{"Test::/a", "Test::/a/b"}, deleted)
	}

	var m updater
	m.On("Update").Return("hello", nil)
	suite.getOrUpdate("Test::/", m.Update)
	suite.getOrUpdate("Test::/a", m.Update)
	suite.getOrUpdate("Test::/a/b", m.Update)
	suite.getOrUpdate("Test::/ab", m.Update)
	m.AssertNumberOfCalls(suite.T(), "Update", 2)
	m.AssertExpectations(suite.T())
}

func TestCache(t *testing.T) {
	suite.Run(t, new(CacheTestSuite))
}
