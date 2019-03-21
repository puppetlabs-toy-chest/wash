package plugin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type EntryBaseTestSuite struct {
	suite.Suite
}

func (suite *EntryBaseTestSuite) TestNewEntry() {
	suite.Panics(
		func() { NewEntry("") },
		"plugin.NewEntry: received an empty name",
	)
	suite.Panics(
		func() { NewEntry("/foo/") },
		"plugin.NewEntry: received a name containing a /",
	)

	e := NewEntry("foo")
	assertOpTTL := func(op actionOpCode, opName string, expectedTTL time.Duration) {
		actualTTL := e.getTTLOf(op)
		suite.Equal(
			expectedTTL,
			actualTTL,
			"expected the TTL of %v to be %v, but got %v instead",
			opName,
			expectedTTL,
			actualTTL,
		)
	}

	suite.Equal("foo", e.Name())
	assertOpTTL(List, "List", 15*time.Second)
	assertOpTTL(Open, "Open", 15*time.Second)
	assertOpTTL(Metadata, "Metadata", 15*time.Second)

	e.setID("/foo")
	suite.Equal("/foo", e.id())

	e.SetTTLOf(List, 40*time.Second)
	assertOpTTL(List, "List", 40*time.Second)

	e.TurnOffCachingFor(List)
	assertOpTTL(List, "List", -1)

	e.TurnOffCaching()
	assertOpTTL(Open, "Open", -1)
	assertOpTTL(Metadata, "Metadata", -1)

	ctx := context.Background()

	attr, err := e.Attr(ctx)
	if suite.NoError(err) {
		suite.Equal(Attributes{}, attr)
	}

	e.Ctime = time.Now()
	attr, err = e.Attr(ctx)
	if suite.NoError(err) {
		expectedAttributes := Attributes{
			Ctime: e.Ctime,
			Mtime: e.Ctime,
			Atime: e.Ctime,
		}

		suite.Equal(expectedAttributes, attr)
	}
}

func TestEntryBase(t *testing.T) {
	suite.Run(t, new(EntryBaseTestSuite))
}
