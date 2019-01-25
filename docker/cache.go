package docker

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/puppetlabs/wash/log"
)

func (cli *Client) cachedContainerList(ctx context.Context) ([]types.Container, error) {
	entry, err := cli.Get("ContainerList")
	var containers []types.Container
	if err == nil {
		log.Debugf("Cache hit in /docker")
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&containers)
	} else {
		log.Debugf("Cache miss in /docker")
		containers, err = cli.ContainerList(ctx, types.ContainerListOptions{})
		if err != nil {
			return nil, err
		}

		var data bytes.Buffer
		enc := gob.NewEncoder(&data)
		if err := enc.Encode(&containers); err != nil {
			return nil, err
		}
		cli.Set("ContainerList", data.Bytes())
		cli.updated = time.Now()
	}
	return containers, err
}

func (cli *Client) cachedContainerInspect(ctx context.Context, name string) (*types.ContainerJSON, error) {
	entry, err := cli.Get(name)
	var container types.ContainerJSON
	if err == nil {
		log.Debugf("Cache hit in /docker/%v", name)
		rdr := bytes.NewReader(entry)
		err = json.NewDecoder(rdr).Decode(&container)
	} else {
		log.Debugf("Cache miss in /docker/%v", name)
		var raw []byte
		container, raw, err = cli.ContainerInspectWithRaw(ctx, name, true)
		if err != nil {
			return nil, err
		}

		cli.Set(name, raw)
	}

	return &container, err
}

func (cli *Client) cachedContainerInspectRaw(ctx context.Context, name string) ([]byte, error) {
	entry, err := cli.Get(name)
	if err == nil {
		log.Debugf("Cache hit in /docker/%v", name)
		return entry, nil
	}

	log.Debugf("Cache miss in /docker/%v", name)
	_, raw, err := cli.ContainerInspectWithRaw(ctx, name, true)
	if err != nil {
		return nil, err
	}

	cli.Set(name, raw)
	return raw, nil
}
