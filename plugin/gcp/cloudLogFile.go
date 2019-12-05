package gcp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"google.golang.org/api/logging/v2"
	"google.golang.org/api/option"
)

type cloudLogEntryField struct {
	name     string
	accessor func(*logging.LogEntry) string
}

// These are some common log entry fields
var severityField = cloudLogEntryField{"level", func(e *logging.LogEntry) string {
	if len(e.Severity) <= 0 {
		return "DEFAULT"
	}
	return e.Severity
}}
var timestampField = cloudLogEntryField{"time_utc", func(e *logging.LogEntry) string { return e.Timestamp }}
var msgField = cloudLogEntryField{"log", func(e *logging.LogEntry) string { return e.TextPayload }}

type cloudLogFile struct {
	service   *logging.Service
	fields    []cloudLogEntryField
	projectID string
	filter    string
}

func newCloudLogFile(
	ctx context.Context,
	client *http.Client,
	fields []cloudLogEntryField,
	projectID string,
	filter string,
) (*cloudLogFile, error) {
	svc, err := logging.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	clf := &cloudLogFile{
		service:   svc,
		fields:    fields,
		projectID: projectID,
		filter:    filter,
	}
	return clf, nil
}

func (clf *cloudLogFile) Read(ctx context.Context) ([]byte, error) {
	// 1000 matches gcloud's upper limit for fetching logs
	entries, err := clf.fetchEntries(ctx, 1000, "")
	if err != nil {
		return nil, err
	}
	var headers []cmdutil.ColumnHeader
	for _, field := range clf.fields {
		headers = append(
			headers,
			cmdutil.ColumnHeader{ShortName: field.name, FullName: strings.ToUpper(field.name)},
		)
	}
	activity.Record(ctx, "HEADERS: %v, ENTRIES: %v", headers, entries)
	table := cmdutil.NewTableWithHeaders(headers, entries)
	return []byte(table.Format()), nil
}

// Note that we use afterTimestamp instead of pageToken because the latter doesn't work well with
// Stream. Specifically, pageToken will reset to the beginning of the log stream if no new entries
// have been added.
func (clf *cloudLogFile) fetchEntries(ctx context.Context, numEntries int64, afterTimestamp string) ([][]string, error) {
	filter := clf.filter
	if len(afterTimestamp) > 0 {
		filter += fmt.Sprintf(" AND timestamp>\"%v\"", afterTimestamp)
	}
	listEntriesReq := &logging.ListLogEntriesRequest{
		ProjectIds: []string{clf.projectID},
		Filter:     filter,
		// We sort by descending order so that the most recent N log entries are
		// returned.
		OrderBy:  "timestamp desc",
		PageSize: numEntries,
	}
	resp, err := clf.service.Entries.List(listEntriesReq).Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	activity.Record(ctx, "Received %v entries", len(resp.Entries))
	entries := make([][]string, len(resp.Entries))
	endIx := len(entries) - 1
	for curIx, logEntry := range resp.Entries {
		var entry []string
		for _, field := range clf.fields {
			entry = append(entry, field.accessor(logEntry))
		}
		// The entries are in descending order so we insert them in reverse order.
		// This ensures that the final result is in ascending order.
		entries[endIx-curIx] = entry
	}
	return entries, nil
}

func (clf *cloudLogFile) Stream(ctx context.Context) (io.ReadCloser, error) {
	return clf.newStreamer(ctx)
}

func (clf *cloudLogFile) newStreamer(ctx context.Context) (*cloudLogFileStreamer, error) {
	s := &cloudLogFileStreamer{
		ctx: ctx,
		clf: clf,
	}
	if err := s.fetchEntries(); err != nil {
		return nil, err
	}
	activity.Record(ctx, "Successfully created the cloud log file streamer")
	return s, nil
}

type cloudLogFileStreamer struct {
	ctx            context.Context
	currentEntries []byte
	afterTimestamp string
	clf            *cloudLogFile
}

func (s *cloudLogFileStreamer) Read(p []byte) (n int, err error) {
	for {
		if len(s.currentEntries) > 0 {
			break
		}
		time.Sleep(2 * time.Second)
		if s.closed() {
			return 0, io.EOF
		}
		activity.Record(s.ctx, "Fetching the next set of cloud logs to stream. After timestamp: %v", s.afterTimestamp)
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

func (s *cloudLogFileStreamer) Close() error {
	// s is closed when the context is cancelled, so this can noop
	return nil
}

func (s *cloudLogFileStreamer) closed() bool {
	select {
	case <-s.ctx.Done():
		return true
	default:
		return false
	}
}

func (s *cloudLogFileStreamer) fetchEntries() error {
	entries, err := s.clf.fetchEntries(s.ctx, 10, s.afterTimestamp)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		s.currentEntries = []byte(cmdutil.NewTable(entries...).Format())
		s.afterTimestamp = entries[len(entries)-1][2]
	}
	return nil
}
