package gcp

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

type cloudFunctionLogService struct {
	*logging.Service
	projectID    string
	region       string
	functionName string
}

type cloudFunctionLog struct {
	plugin.EntryBase
	service cloudFunctionLogService
}

func newCloudFunctionLog(ctx context.Context, service cloudFunctionsProjectService, region string, functionName string) (*cloudFunctionLog, error) {
	svc, err := logging.NewService(ctx, option.WithHTTPClient(service.client))
	if err != nil {
		return nil, err
	}
	cfl := &cloudFunctionLog{
		EntryBase: plugin.NewEntry("log"),
		service:   cloudFunctionLogService{svc, service.projectID, region, functionName},
	}
	return cfl, nil
}

func (cfl *cloudFunctionLog) Open(ctx context.Context) (plugin.SizedReader, error) {
	// 1000 matches gcloud's upper limit for fetching logs
	entries, err := cfl.fetchEntries(ctx, 1000, "")
	if err != nil {
		return nil, err
	}
	table := cmdutil.NewTableWithHeaders(
		[]cmdutil.ColumnHeader{
			{ShortName: "level", FullName: "LEVEL"},
			{ShortName: "execution_id", FullName: "EXECUTION_ID"},
			{ShortName: "time_utc", FullName: "TIME_UTC"},
			{ShortName: "log", FullName: "LOG"},
		},
		entries,
	)
	return strings.NewReader(table.Format()), nil
}

// Note that we use afterTimestamp instead of pageToken because the latter doesn't work well with
// Stream. Specifically, pageToken will reset to the beginning of the log stream if no new entries
// have been added.
func (cfl *cloudFunctionLog) fetchEntries(ctx context.Context, numEntries int64, afterTimestamp string) ([][]string, error) {
	filter := fmt.Sprintf(
		"logName:\"cloud-functions\" AND resource.type=\"cloud_function\" AND resource.labels.region=\"%v\" AND resource.labels.function_name=\"%v\"",
		cfl.service.region,
		cfl.service.functionName,
	)
	if len(afterTimestamp) > 0 {
		filter += fmt.Sprintf(" AND timestamp>\"%v\"", afterTimestamp)
	}
	listEntriesReq := &logging.ListLogEntriesRequest{
		ProjectIds: []string{cfl.service.projectID},
		Filter:     filter,
		// We sort by descending order so that the most recent N log entries are
		// returned.
		OrderBy:  "timestamp desc",
		PageSize: numEntries,
	}
	resp, err := cfl.service.Entries.List(listEntriesReq).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	activity.Record(ctx, "Received %v entries", len(resp.Entries))
	entries := make([][]string, len(resp.Entries))
	endIx := len(entries) - 1
	for curIx, entry := range resp.Entries {
		// The entries are in descending order so we insert them in reverse order.
		// This ensures that the final result is in ascending order.
		//
		// Note that we leave the severity and timestamp as-is to make them easier
		// to parse with existing log libraries. This is slightly different from
		// `gcloud function logs read <function_name>`
		entries[endIx-curIx] = []string{entry.Severity, entry.Labels["execution_id"], entry.Timestamp, entry.TextPayload}
	}
	return entries, nil
}

func (cfl *cloudFunctionLog) Stream(ctx context.Context) (io.ReadCloser, error) {
	return cfl.newStreamer(ctx)
}

func (cfl *cloudFunctionLog) newStreamer(ctx context.Context) (*cloudFunctionLogStreamer, error) {
	s := &cloudFunctionLogStreamer{
		ctx: ctx,
		cfl: cfl,
	}
	if err := s.fetchEntries(); err != nil {
		return nil, err
	}
	activity.Record(ctx, "Successfully created the cloud function log streamer")
	return s, nil
}

func (cfl *cloudFunctionLog) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(cfl, "log").
		IsSingleton().
		SetDescription(cloudFunctionLogDescription)
}

const cloudFunctionLogDescription = `
This is a cloud function's log. Each line is formatted as
    LEVEL EXECUTION_ID TIME_UTC LOG
`

type cloudFunctionLogStreamer struct {
	ctx            context.Context
	currentEntries []byte
	afterTimestamp string
	cfl            *cloudFunctionLog
}

func (s *cloudFunctionLogStreamer) Read(p []byte) (n int, err error) {
	for {
		if len(s.currentEntries) > 0 {
			break
		}
		time.Sleep(2 * time.Second)
		if s.closed() {
			return 0, io.EOF
		}
		activity.Record(s.ctx, "Fetching the next set of cloud function logs to stream. After timestamp: %v", s.afterTimestamp)
		if err := s.fetchEntries(); err != nil {
			return 0, err
		}
	}
	if s.closed() {
		return 0, io.EOF
	}
	numCopied := copy(p, s.currentEntries)
	s.currentEntries = s.currentEntries[numCopied:]
	return numCopied, nil

}

func (s *cloudFunctionLogStreamer) Close() error {
	// s is closed when the context is cancelled, so this can noop
	return nil
}

func (s *cloudFunctionLogStreamer) closed() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *cloudFunctionLogStreamer) fetchEntries() error {
	entries, err := s.cfl.fetchEntries(s.ctx, 10, s.afterTimestamp)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		s.currentEntries = []byte(cmdutil.NewTable(entries...).Format())
		s.afterTimestamp = entries[len(entries)-1][2]
	}
	return nil
}
