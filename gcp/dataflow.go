package gcp

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"sort"
	"time"

	"github.com/puppetlabs/wash/log"
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

func (cli *service) cachedDataflowJobs(ctx context.Context, c *dataflow.Service) ([]string, error) {
	key := cli.proj + "/dataflow/" + cli.name
	entry, err := cli.cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit in /gcp")
		var jobs []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&jobs)
		return jobs, err
	}

	log.Debugf("Cache miss in /gcp")
	projJobSvc := dataflow.NewProjectsJobsService(c)
	projJobsResp, err := projJobSvc.List(cli.proj).Do()
	if err != nil {
		return nil, err
	}

	jobs := make([]string, len(projJobsResp.Jobs))
	for i, job := range projJobsResp.Jobs {
		jobs[i] = job.Name
	}
	sort.Strings(jobs)

	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(&jobs); err != nil {
		return nil, err
	}
	cli.cache.Set(key, data.Bytes())
	cli.updated = time.Now()
	return jobs, nil
}
