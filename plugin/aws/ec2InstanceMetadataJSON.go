package aws

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/puppetlabs/wash/plugin"
)

// ec2InstanceMetadataJSON represents an EC2 instance's
// metadata.json file
type ec2InstanceMetadataJSON struct {
	plugin.EntryBase
	inst *ec2Instance
}

func newEC2InstanceMetadataJSON(ctx context.Context, inst *ec2Instance) (*ec2InstanceMetadataJSON, error) {
	im := &ec2InstanceMetadataJSON{
		EntryBase: plugin.NewEntry("metadata.json"),
		inst:      inst,
	}
	im.DisableDefaultCaching()

	meta, err := im.Metadata(ctx)
	if err != nil {
		return nil, err
	}

	attr := plugin.EntryAttributes{}
	attr.SetSize(uint64(meta["Size"].(int64)))
	attr.SetMeta(meta)
	im.SetAttributes(attr)

	return im, nil
}

func (im *ec2InstanceMetadataJSON) Metadata(ctx context.Context) (plugin.EntryMetadata, error) {
	content, err := im.Open(ctx)
	if err != nil {
		return nil, err
	}

	return plugin.EntryMetadata{
		"Size": content.Size(),
	}, nil
}

func (im *ec2InstanceMetadataJSON) Open(ctx context.Context) (plugin.SizedReader, error) {
	meta, err := plugin.CachedMetadata(ctx, im.inst)
	if err != nil {
		return nil, err
	}

	prettyMeta, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(prettyMeta), nil
}
