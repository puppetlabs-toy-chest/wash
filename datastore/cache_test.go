package datastore

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MemCacheTestSuite struct {
	suite.Suite
	mem *MemCache
}

func (suite *MemCacheTestSuite) SetupTest() {
	suite.mem = NewMemCache()
}

func (suite *MemCacheTestSuite) TestClearCache() {
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
