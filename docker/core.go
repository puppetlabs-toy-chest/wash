package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
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

// List returns a list of running container objects.
func (cli *Client) List() ([]types.Container, error) {
	return cli.ContainerList(context.Background(), types.ContainerListOptions{})
}
