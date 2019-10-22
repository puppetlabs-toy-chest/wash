package plugin

import (
	"strconv"
	"sync"
	"testing"
	"time"

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

func (suite *EntryMapTestSuite) TestConcurrentReadWrite() {
	m := newEntryMap()

	var wg sync.WaitGroup
	var startCh = make(chan struct{})
	var doneCh = make(chan struct{})

	// Load, Delete, and Range all acquire locks to avoid a concurrent
	// read/write panic. Thus, this test launches goroutines that invoke
	// each method concurrently. Idea is that if any one of those methods
	// fail to acquire a lock (including the right lock, like a Write lock
	// for Delete), then this test will panic. NumGoroutinesPerMethod is
	// arbitrary, idea is it should be high enough that we can detect a panic
	// but low enough that the test won't take too long.
	//
	// The test passes if all the goroutines successfully return.
	//
	// NOTE: Testing Len() is a bit tricky because a concurrent len(mp) and
	// delete(mp) does not cause a read/write panic. Since its implementation is
	// simple enough, we omit Len() to avoid further complicating the tests.
	NumGoroutinesPerMethod := 40
	for i := 0; i < NumGoroutinesPerMethod; i++ {
		key := strconv.Itoa(i)
		m.mp[key] = nil

		// Load
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startCh
			m.Load(key)
		}(i)

		// Delete
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startCh
			m.Delete(key)
		}(i)

		// Range
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-startCh
			m.Range(func(_ string, _ Entry) bool {
				return false
			})
		}(i)
	}
	time.AfterFunc(5*time.Second, func() {
		select {
		case <-doneCh:
			// Pass-thru
		default:
			panic("goroutines did not successfully return after 5 seconds. Did you forget to release a lock?")
		}
	})

	// Start the goroutines and wait for them to finish.
	close(startCh)
	wg.Wait()
	close(doneCh)
}

func TestEntryMap(t *testing.T) {
	suite.Run(t, new(EntryMapTestSuite))
}
