package gcp

import (
	"context"
	"io"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	dataflow "google.golang.org/api/dataflow/v1b3"
)

type dataflowJob struct {
	name    string
	id      string
	client  *dataflow.Service
	updated time.Time
	*service
}

// Constructs a dataflowJob from id, which combines name and job id.
func newDataflowJob(id string, client *dataflow.Service, svc *service) *dataflowJob {
	name, id := datastore.SplitCompositeString(id)
	return &dataflowJob{name, id, client, time.Now(), svc}
}

// String returns a unique representation of the dataflow job.
func (cli *dataflowJob) String() string {
	return cli.service.String() + "/" + cli.Name()
}

// Returns the dataflow job name.
func (cli *dataflowJob) Name() string {
	return cli.name
}

// Attr returns attributes of the named resource.
func (cli *dataflowJob) Attr(ctx context.Context) (*plugin.Attributes, error) {
	if v, ok := cli.reqs.Load(cli.name); ok {
		buf := v.(*datastore.StreamBuffer)
		return &plugin.Attributes{Mtime: buf.LastUpdate(), Size: uint64(buf.Size())}, nil
	}

	// Prefetch content for next time.
	go plugin.PrefetchOpen(cli)

	return &plugin.Attributes{Mtime: cli.updated}, nil
}

// Xattr returns a map of extended attributes.
func (cli *dataflowJob) Xattr(ctx context.Context) (map[string][]byte, error) {
	data, err := cli.cache.CachedJSON(cli.String(), func() ([]byte, error) {
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

type dataflowReader struct {
	*dataflow.ProjectsJobsMessagesListCall
	overflow []*dataflow.JobMessage
	eof      bool
}

func (rdr *dataflowReader) consume(p []byte) ([]byte, int) {
	consumed, n := 0, 0
	for _, msg := range rdr.overflow {
		msgLen := len(msg.Time) + len(msg.MessageImportance) + len(msg.MessageText) + 3
		if msgLen > len(p) {
			break
		}
		copy(p, msg.Time+" "+msg.MessageImportance+" "+msg.MessageText+"\n")
		p = p[msgLen:]
		n += msgLen
		consumed++
	}
	rdr.overflow = rdr.overflow[consumed:]
	return p, n
}

func (rdr *dataflowReader) Read(p []byte) (n int, err error) {
	// If there was data left over, consume it. If any remains after filling the buffer, return.
	var read int
	if len(rdr.overflow) > 0 {
		p, read = rdr.consume(p)
		n += read
		if len(rdr.overflow) > 0 {
			return
		}
	}

	// If EOF was reached on a previous call, return that. We only reach this point if all overflow
	// has been consumed. Includes the number of bytes processing remaining overflow.
	if rdr.eof {
		err = io.EOF
		return
	}

	// Keep reading pages from the API as needed to fill the buffer. Stash overflow.
	var resp *dataflow.ListJobMessagesResponse
	for {
		resp, err = rdr.Do()
		if err != nil {
			return
		}

		// Process response
		rdr.overflow = resp.JobMessages
		p, read = rdr.consume(p)
		n += read

		// Setup the next read. If NextPageToken was empty, mark EOF.
		rdr.PageToken(resp.NextPageToken)
		if resp.NextPageToken == "" {
			rdr.eof = true
		}

		// If the buffer is full or there's no more data to read, return.
		if len(rdr.overflow) > 0 || rdr.eof {
			return
		}
	}
}

func (rdr *dataflowReader) Close() error {
	return nil
}

func (cli *dataflowJob) readLog() (io.ReadCloser, error) {
	lister := dataflow.NewProjectsJobsMessagesService(cli.client).List(cli.proj, cli.id)
	return &dataflowReader{ProjectsJobsMessagesListCall: lister}, nil
}

// Open subscribes to a dataflow job and reads new messages.
func (cli *dataflowJob) Open(ctx context.Context) (plugin.IFileBuffer, error) {
	buf := datastore.NewBuffer(cli.name, nil)
	if v, ok := cli.reqs.LoadOrStore(cli.name, buf); ok {
		buf = v.(*datastore.StreamBuffer)
	}

	buffered := make(chan bool)
	go func() {
		buf.Stream(cli.readLog, buffered)
	}()
	// Wait for some output to buffer.
	<-buffered

	return buf, nil
}

// Returns an array where every even entry is a job name and the following entry is its id.
func (cli *service) cachedDataflowJobs(c *dataflow.Service) ([]string, error) {
	return cli.cache.CachedStrings(cli.String(), func() ([]string, error) {
		projJobSvc := dataflow.NewProjectsJobsService(c)
		projJobsResp, err := projJobSvc.List(cli.proj).Do()
		if err != nil {
			return nil, err
		}

		jobs := make([]string, len(projJobsResp.Jobs))
		for i, job := range projJobsResp.Jobs {
			jobs[i] = datastore.MakeCompositeString(job.Name, job.Id)
		}
		cli.updated = time.Now()
		return jobs, nil
	})
}
