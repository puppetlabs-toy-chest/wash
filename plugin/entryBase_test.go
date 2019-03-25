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

	e.DisableCachingFor(List)
	assertOpTTL(List, "List", -1)

	e.DisableDefaultCaching()
	assertOpTTL(Open, "Open", -1)
	assertOpTTL(Metadata, "Metadata", -1)

	ctx := context.Background()
	attr, err := e.Attr(ctx)
	if suite.NoError(err) {
		expectedAttr := Attributes{
			Size: SizeUnknown,
		}

		suite.Equal(expectedAttr, attr)
	}
}

func (suite *EntryBaseTestSuite) TestSetSlashReplacementChar() {
	e := NewEntry("foo/bar")

	suite.Panics(
		func() { e.SetSlashReplacementChar('/') },
		"e.SetSlashReplacementChar called with '/'",
	)

	e.SetSlashReplacementChar(':')
	suite.Equal(e.slashReplacementChar(), ':')
}

func TestEntryBase(t *testing.T) {
	suite.Run(t, new(EntryBaseTestSuite))
}
