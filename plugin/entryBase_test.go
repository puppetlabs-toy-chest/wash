package plugin

import (
	"context"
	"os"
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

func (suite *EntryBaseTestSuite) TestSync() {
	e := NewEntry("foo")

	e.Sync(CtimeAttr(), "CreationDate")
	if suite.Equal(1, len(e.syncedAttrs)) {
		syncedAttr := e.syncedAttrs[0]
		suite.Equal(CtimeAttr().name, syncedAttr.name)
		suite.Equal("CreationDate", syncedAttr.key)
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

func (suite *EntryBaseTestSuite) TestSyncAttributesWithNoErrors() {
	initialAttr := EntryAttributes{}
	initialAttr.SetCtime(time.Now())
	initialAttr.SetMtime(time.Now())

	e := NewEntry("foo")
	e.SetInitialAttributes(initialAttr)
	e.Sync(MtimeAttr(), "LastModified")
	e.Sync(SizeAttr(), "Size")

	meta := EntryMetadata{
		"LastModified": time.Now(),
		"Size":         uint64(15),
	}
	err := e.syncAttributesWith(meta)
	if suite.NoError(err) {
		expectedAttr := initialAttr.toMap()
		expectedAttr[MtimeAttr().name] = meta["LastModified"]
		expectedAttr[SizeAttr().name] = meta["Size"]
		expectedAttr["meta"] = meta

		suite.Equal(expectedAttr, e.attr.toMap())
	}
}

func (suite *EntryBaseTestSuite) TestSyncAttributesWithErrors() {
	initialAttr := EntryAttributes{}
	initialAttr.SetMode(os.FileMode(0))
	initialAttr.SetSize(10)

	e := NewEntry("foo")
	e.SetInitialAttributes(initialAttr)
	e.Sync(ModeAttr(), "Mode")
	e.Sync(SizeAttr(), "Size")

	meta := EntryMetadata{
		"Mode": "badmode",
		"Size": int64(-1),
	}
	err := e.syncAttributesWith(meta)
	suite.Regexp("sync.*mode.*attr.*sync.*size.*attr", err)
}

func TestEntryBase(t *testing.T) {
	suite.Run(t, new(EntryBaseTestSuite))
}
