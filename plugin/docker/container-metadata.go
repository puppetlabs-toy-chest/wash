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

func (cm *containerMetadata) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	content, err := cm.Open(ctx)
	if err != nil {
		return nil, err
	}

	return plugin.EntryMetadata{
		"Size": content.Size(),
	}, nil
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
