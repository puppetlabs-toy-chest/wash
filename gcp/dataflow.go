package gcp

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/plugin"
	dataflow "google.golang.org/api/dataflow/v1b3"
)

type dataflowJob struct {
	name   string
	client *dataflow.Service
	*service
}

// String returns a printable representation of the dataflow job.
func (cli *dataflowJob) String() string {
	return fmt.Sprintf("gcp/%v/dataflow/job/%v", cli.proj, cli.name)
}

// Returns the dataflow job name.
func (cli *dataflowJob) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *dataflowJob) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if buf, ok := cli.reqs[cli.name]; ok {
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size()), Valid: validDuration}, nil
	}

	return &plugin.Attributes{Mtime: cli.updated, Valid: validDuration}, nil
}

// Xattr returns a map of extended attributes.
func (cli *dataflowJob) Xattr(ctx context.Context) (map[string][]byte, error) {
	// TODO: get dataflow metadata, https://godoc.org/google.golang.org/api/dataflow/v1b3#Job
	return nil, plugin.ENOTSUP
}

// Open subscribes to a dataflow job and reads new messages.
func (cli *dataflowJob) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: read dataflow logs, https://godoc.org/google.golang.org/api/dataflow/v1b3#ProjectsJobsMessagesService.List
	return nil, plugin.ENOTSUP
}
