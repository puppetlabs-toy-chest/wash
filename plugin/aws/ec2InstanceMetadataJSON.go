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

func newEC2InstanceMetadataJSON(inst *ec2Instance) *ec2InstanceMetadataJSON {
	return &ec2InstanceMetadataJSON{
		EntryBase: plugin.NewEntry("metadata.json"),
		inst:      inst,
	}
}

func (im *ec2InstanceMetadataJSON) Open(ctx context.Context) (plugin.SizedReader, error) {
	metadata, err := plugin.CachedMetadata(ctx, im.inst)
	if err != nil {
		return nil, err
	}

	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(content), nil
}
