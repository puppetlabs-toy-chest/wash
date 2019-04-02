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

func (suite *EntryBaseTestSuite) assertOpTTL(e EntryBase, op defaultOpCode, opName string, expectedTTL time.Duration) {
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

func (suite *EntryBaseTestSuite) TestNewEntry() {
	suite.Panics(
		func() { NewEntry("") },
		"plugin.NewEntry: received an empty name",
	)

	initialAttr := EntryAttributes{}
	initialAttr.SetCtime(time.Now())
	e := NewEntry("foo")

	e.SetInitialAttributes(initialAttr)
	suite.Equal(initialAttr, e.attr)

	suite.Equal("foo", e.Name())
	suite.assertOpTTL(e, ListOp, "List", 15*time.Second)
	suite.assertOpTTL(e, OpenOp, "Open", 15*time.Second)
	suite.assertOpTTL(e, MetadataOp, "Metadata", 15*time.Second)

	e.setID("/foo")
	suite.Equal("/foo", e.id())

	e.SetTTLOf(ListOp, 40*time.Second)
	suite.assertOpTTL(e, ListOp, "List", 40*time.Second)

	e.DisableCachingFor(ListOp)
	suite.assertOpTTL(e, ListOp, "List", -1)

	e.DisableDefaultCaching()
	suite.assertOpTTL(e, OpenOp, "Open", -1)
	suite.assertOpTTL(e, MetadataOp, "Metadata", -1)
}

func (suite *EntryBaseTestSuite) TestMetadata() {
	e := NewEntry("foo")

	meta, err := e.Metadata(context.Background())
	if suite.NoError(err) {
		suite.Equal(EntryMetadata{}, meta)
	}
	suite.assertOpTTL(e, MetadataOp, "Metadata", -1)
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
