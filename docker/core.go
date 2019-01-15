package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/puppetlabs/wash/plugin"
)

// Client is a docker client.
type Client struct {
	*client.Client
}

// Create a new docker client.
func Create() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Client{cli}, nil
}

// Find container by ID.
func (cli *Client) Find(name string) (*plugin.Entry, error) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if container.ID == name {
			return &plugin.Entry{Client: cli, Name: container.ID}, nil
		}
	}
	return nil, plugin.ENOENT
}

// List all running containers as files.
func (cli *Client) List() ([]plugin.Entry, error) {
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	keys := make([]plugin.Entry, len(containers))
	for i, container := range containers {
		keys[i] = plugin.Entry{Client: cli, Name: container.ID}
	}
	return keys, nil
}
