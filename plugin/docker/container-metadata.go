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

func newContainerMetadata(container *container) *containerMetadata {
	cm := &containerMetadata{
		EntryBase: plugin.NewEntry("metadata.json"),
	}
	cm.DisableDefaultCaching()
	cm.container = container
	return cm
}

func (cm *containerMetadata) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cm, "metadata.json").IsSingleton()
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
