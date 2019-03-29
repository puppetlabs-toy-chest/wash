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
	content, err := cm.content(ctx)
	if err != nil {
		return nil, err
	}

	return plugin.EntryMetadata{
		"Size": uint64(len(content)),
	}, nil
}

func (cm *containerMetadata) Open(ctx context.Context) (plugin.SizedReader, error) {
	content, err := cm.content(ctx)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), err
}

func (cm *containerMetadata) content(ctx context.Context) ([]byte, error) {
	meta, err := plugin.CachedMetadata(ctx, cm.container)
	if err != nil {
		return nil, err
	}

	prettyMeta, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}

	return prettyMeta, nil
}
