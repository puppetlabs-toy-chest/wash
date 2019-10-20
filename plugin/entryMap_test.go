package plugin

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EntryMapTestSuite struct {
	suite.Suite
}

func (suite *EntryMapTestSuite) TestLoad_Delete_Len() {
	m := newEntryMap()

	_, ok := m.Load("foo")
	suite.False(ok)

	suite.Equal(0, m.Len())

	expectedEntry := newCacheTestsMockEntry("foo")
	m.mp["foo"] = expectedEntry

	actualEntry, ok := m.Load("foo")
	suite.Equal(expectedEntry, actualEntry)
	suite.True(ok)

	suite.Equal(1, m.Len())

	m.Delete("foo")
	_, ok = m.Load("foo")
	suite.False(ok)

	suite.Equal(0, m.Len())
}

func (suite *EntryMapTestSuite) TestRange() {
	m := newEntryMap()

	entries := []Entry{
		newCacheTestsMockEntry("foo"),
		newCacheTestsMockEntry("bar"),
	}
	for _, entry := range entries {
		m.mp[CName(entry)] = entry
	}

	// Test that iteration works
	entryMap := make(map[string]Entry)
	m.Range(func(cname string, entry Entry) bool {
		entryMap[cname] = entry
		return true
	})
	suite.Equal(m.Map(), entryMap)

	// Test that break works
	count := 0
	m.Range(func(cname string, entry Entry) bool {
		if cname == "foo" {
			return false
		}
		count++
		return true
	})
	// <= 1 is to account for the fact that Go map's iteration order
	// is random.
	suite.True(count <= 1)
}

func TestEntryMap(t *testing.T) {
	suite.Run(t, new(EntryMapTestSuite))
}
