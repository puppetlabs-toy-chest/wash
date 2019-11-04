package volume

import (
	"context"
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type coreTestSuite struct {
	suite.Suite
}

func (s *coreTestSuite) TestDeleteNode_ReturnsVolumeDeleteError() {
	ctx := context.Background()
	mockImpl := &mockDirEntry{EntryBase: plugin.NewEntry("foo")}
	path := "bar"

	expectedErr := fmt.Errorf("failed to delete")
	mockImpl.On("VolumeDelete", ctx, path).Return(false, expectedErr)

	_, err := deleteNode(ctx, mockImpl, path, &dirMap{})
	s.EqualError(expectedErr, err.Error())
}

func (s *coreTestSuite) TestDeleteNode_NodeDeletionInProgress_LeavesDirMapAlone() {
	ctx := context.Background()
	mockImpl := &mockDirEntry{EntryBase: plugin.NewEntry("foo")}
	path := "bar/baz"
	dirMap := &dirMap{
		mp: map[string]Children{
			"bar/baz": map[string]plugin.EntryAttributes{
				"baz": plugin.EntryAttributes{},
			},
		},
	}

	mockImpl.On("VolumeDelete", ctx, path).Return(false, nil)

	deleted, err := deleteNode(ctx, mockImpl, path, dirMap)
	if s.NoError(err) {
		s.False(deleted)
		s.Contains(dirMap.mp, "bar/baz")
		s.Contains(dirMap.mp["bar/baz"], "baz")
	}
}

func (s *coreTestSuite) TestDeleteNode_DeletedNode_UpdatesDirMap() {
	ctx := context.Background()
	mockImpl := &mockDirEntry{EntryBase: plugin.NewEntry("foo")}
	path := "bar/baz"
	dirMap := &dirMap{
		mp: map[string]Children{
			"bar/baz": map[string]plugin.EntryAttributes{
				"baz": plugin.EntryAttributes{},
			},
		},
	}

	mockImpl.On("VolumeDelete", ctx, path).Return(true, nil)

	deleted, err := deleteNode(ctx, mockImpl, path, dirMap)
	if s.NoError(err) {
		s.True(deleted)
		s.NotContains(dirMap.mp, "bar/baz")
		s.NotContains(dirMap.mp["bar/baz"], "baz")
	}
}

func TestCore(t *testing.T) {
	suite.Run(t, new(coreTestSuite))
}
