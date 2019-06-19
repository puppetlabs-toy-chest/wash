package docker

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/puppetlabs/wash/plugin"
)

type containerMetadata struct {
	plugin.EntryBase
	container *container
}

func containerMetadataBase(forInstance bool) *containerMetadata {
	cm := &containerMetadata{
		EntryBase: plugin.NewEntryBase(),
	}
	cm.
		SetName("metadata.json").
		IsSingleton().
		DisableDefaultCaching()
	return cm
}

func newContainerMetadata(container *container) *containerMetadata {
	cm := containerMetadataBase(true)
	cm.container = container
	return cm
}

func (cm *containerMetadata) Open(ctx context.Context) (plugin.SizedReader, error) {
	metadata, err := plugin.CachedMetadata(ctx, cm.container)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}
