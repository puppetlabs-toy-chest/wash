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

func (suite *EntryBaseTestSuite) TestNewEntryBase() {
	initialAttr := EntryAttributes{}
	initialAttr.SetCtime(time.Now())
	e := NewEntryBase()

	e.SetAttributes(initialAttr)
	suite.Equal(initialAttr, e.attr)

	e.SetName("foo")
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
	e := NewEntryBase()

	meta, err := e.Metadata(context.Background())
	if suite.NoError(err) {
		suite.Equal(JSONObject{}, meta)
	}
	suite.assertOpTTL(e, MetadataOp, "Metadata", -1)
}

func (suite *EntryBaseTestSuite) TestSetSlashReplacer() {
	e := NewEntryBase()

	suite.Panics(
		func() { e.SetSlashReplacer('/') },
		"e.SetSlashReplacer called with '/'",
	)

	e.SetSlashReplacer(':')
	suite.Equal(e.slashReplacer(), ':')
}

func TestEntryBase(t *testing.T) {
	suite.Run(t, new(EntryBaseTestSuite))
}
