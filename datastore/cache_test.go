package datastore

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const anything = "anything"

type MemCacheTestSuite struct {
	suite.Suite
	mem   *MemCache
	thing mock.Mock
}

func (suite *MemCacheTestSuite) SetupTest() {
	suite.mem = NewMemCache()
	suite.thing = mock.Mock{}
}

func (suite *MemCacheTestSuite) update() (interface{}, error) {
	args := suite.thing.Called()
	return args.Get(0), args.Error(1)
}

func (suite *MemCacheTestSuite) validate(item interface{}, err error) {
	if suite.Nil(err) {
		suite.Equal(anything, item)
	}
}

func (suite *MemCacheTestSuite) TestGetOrUpdateNoReset() {
	suite.thing.On("update").Return(anything, nil)

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Second, false, suite.update))
	item, ok := suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Nanosecond, false, suite.update))
	time.Sleep(time.Nanosecond)
	item, ok = suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}
	suite.thing.AssertNumberOfCalls(suite.T(), "update", 1)

	suite.mem.instance.Delete("cat::an entry")
	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Second, false, suite.update))
	item, ok = suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}
	suite.thing.AssertNumberOfCalls(suite.T(), "update", 2)
}

func (suite *MemCacheTestSuite) TestGetOrUpdateExpire() {
	suite.thing.On("update").Return(anything, nil)

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Nanosecond, false, suite.update))
	time.Sleep(time.Nanosecond)
	_, ok := suite.mem.instance.Get("cat::an entry")
	suite.False(ok)

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Second, false, suite.update))
	item, ok := suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}
	suite.thing.AssertNumberOfCalls(suite.T(), "update", 2)
}

func (suite *MemCacheTestSuite) TestGetOrUpdateWithReset() {
	suite.thing.On("update").Return(anything, nil)

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Second, true, suite.update))
	item, ok := suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Nanosecond, true, suite.update))
	time.Sleep(time.Nanosecond)
	_, ok = suite.mem.instance.Get("cat::an entry")
	suite.False(ok)
	suite.thing.AssertNumberOfCalls(suite.T(), "update", 1)

	suite.validate(suite.mem.GetOrUpdate("cat", "an entry", time.Second, true, suite.update))
	item, ok = suite.mem.instance.Get("cat::an entry")
	if suite.True(ok) {
		suite.Equal("anything", item)
	}
	suite.thing.AssertNumberOfCalls(suite.T(), "update", 2)
}

func (suite *MemCacheTestSuite) TestFlush() {
	suite.mem.instance.Set("an entry", struct{}{}, time.Nanosecond)
	time.Sleep(time.Nanosecond)
	suite.mem.instance.SetDefault("another entry", struct{}{})
	suite.mem.Flush()
	suite.Equal(0, suite.mem.instance.ItemCount())
}

func (suite *MemCacheTestSuite) TestDelete() {
	suite.mem.instance.SetDefault("an entry", struct{}{})
	suite.mem.instance.SetDefault("another entry", struct{}{})
	suite.NotNil(suite.mem.instance.Get("an entry"))

	matcher, err := regexp.Compile("^.*n e.*$")
	suite.Nil(err)
	deleted := suite.mem.Delete(matcher)
	suite.Equal([]string{"an entry"}, deleted)

	suite.Nil(suite.mem.instance.Get("an entry"))
	suite.NotNil(suite.mem.instance.Get("another entry"))
}

func TestMemCache(t *testing.T) {
	suite.Run(t, new(MemCacheTestSuite))
}

type MemCacheEvictedTestSuite struct {
	suite.Suite
	mem     *MemCache
	evictor mock.Mock
}

func (suite *MemCacheEvictedTestSuite) evict(s string, i interface{}) {
	suite.evictor.Called(s, i)
}

func (suite *MemCacheEvictedTestSuite) SetupTest() {
	suite.mem = NewMemCacheWithEvicted(suite.evict)
	suite.evictor = mock.Mock{}
}

func (suite *MemCacheEvictedTestSuite) TestFlush() {
	suite.mem.instance.Set("an entry", struct{}{}, time.Nanosecond)
	time.Sleep(time.Nanosecond)
	suite.mem.instance.SetDefault("another entry", struct{}{})

	suite.evictor.On("evict", "an entry", mock.Anything)
	suite.evictor.On("evict", "another entry", mock.Anything)
	suite.mem.Flush()
	suite.Equal(0, suite.mem.instance.ItemCount())
	suite.evictor.AssertExpectations(suite.T())
}

// TODO: Double-check this test to make sure it's testing the right
// thing
func (suite *MemCacheEvictedTestSuite) TestExpired() {
	suite.mem.instance.Set("an entry", struct{}{}, time.Nanosecond)
	time.Sleep(time.Nanosecond)

	_, err := suite.mem.GetOrUpdate("cat", "an entry", time.Second, false, func() (interface{}, error) {
		return nil, errors.New("nope")
	})
	suite.Equal(errors.New("nope"), err)
	if val, ok := suite.mem.instance.Get("cat::an entry"); suite.True(ok) {
		suite.Equal(errors.New("nope"), val)
	}
}

func (suite *MemCacheEvictedTestSuite) TestDelete() {
	suite.mem.instance.SetDefault("an entry", struct{}{})
	suite.mem.instance.SetDefault("another entry", struct{}{})

	matcher, err := regexp.Compile("^.*n e.*$")
	suite.Nil(err)

	suite.evictor.On("evict", "an entry", mock.Anything)
	deleted := suite.mem.Delete(matcher)
	suite.Equal([]string{"an entry"}, deleted)
	suite.evictor.AssertExpectations(suite.T())

	suite.Nil(suite.mem.instance.Get("an entry"))
	suite.NotNil(suite.mem.instance.Get("another entry"))
}

func TestMemCacheEvicted(t *testing.T) {
	suite.Run(t, new(MemCacheEvictedTestSuite))
}
