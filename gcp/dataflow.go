package gcp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	dataflow "google.golang.org/api/dataflow/v1b3"
)

type dataflowJob struct {
	name   string
	id     string
	client *dataflow.Service
	*service
}

// Constructs a dataflowJob from id, which combines name and job id.
func newDataflowJob(id string, client *dataflow.Service, svc *service) *dataflowJob {
	name, id := splitDataflowID(id)
	return &dataflowJob{name, id, client, svc}
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
	data, err := datastore.CachedJSON(cli.cache, cli.String(), func() ([]byte, error) {
		projJobSvc := dataflow.NewProjectsJobsService(cli.client)
		job, err := projJobSvc.Get(cli.proj, cli.id).Do()
		if err != nil {
			return nil, err
		}
		return job.MarshalJSON()
	})
	if err != nil {
		return nil, err
	}
	return plugin.JSONToJSONMap(data)
}

// Open subscribes to a dataflow job and reads new messages.
func (cli *dataflowJob) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	// TODO: read dataflow logs, https://godoc.org/google.golang.org/api/dataflow/v1b3#ProjectsJobsMessagesService.List
	return nil, plugin.ENOTSUP
}

// Returns an array where every even entry is a job name and the following entry is its id.
func (cli *service) cachedDataflowJobs(c *dataflow.Service) ([]string, error) {
	return datastore.CachedStrings(cli.cache, cli.String(), func() ([]string, error) {
		projJobSvc := dataflow.NewProjectsJobsService(c)
		projJobsResp, err := projJobSvc.List(cli.proj).Do()
		if err != nil {
			return nil, err
		}

		jobs := make([]string, len(projJobsResp.Jobs))
		for i, job := range projJobsResp.Jobs {
			jobs[i] = job.Name + "/" + job.Id
		}
		cli.updated = time.Now()
		return jobs, nil
	})
}

func searchDataflowJob(jobs []string, name string) (string, bool) {
	idx := sort.Search(len(jobs), func(i int) bool {
		x, _ := splitDataflowID(jobs[i])
		return x >= name
	})
	if idx < len(jobs) {
		x, _ := splitDataflowID(jobs[idx])
		if x == name {
			return jobs[idx], true
		}
	}
	return "", false
}

func splitDataflowID(id string) (string, string) {
	// name is required to match [a-z]([-a-z0-9]{0,38}[a-z0-9])?, and id can additionally
	// include underscores. Use '/' as a separator.
	tokens := strings.Split(id, "/")
	if len(tokens) != 2 {
		panic("newDataflowJob given an invalid name/id pair")
	}
	return tokens[0], tokens[1]
}
