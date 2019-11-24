package gcp

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/logging/v2"
)

type cloudRunServiceLog struct {
	plugin.EntryBase
	*cloudLogFile
}

func newCloudRunServiceLog(ctx context.Context, apiService cloudRunProjectAPIService, region string, serviceName string) (*cloudRunServiceLog, error) {
	fields := []cloudLogEntryField{
		{"level", func(e *logging.LogEntry) string { return e.Severity }},
		// Return only the first 16 characters of the instance ID to make the output readable
		{"instance_id", func(e *logging.LogEntry) string { return e.Labels["instanceId"][0:15] }},
		{"time_utc", func(e *logging.LogEntry) string { return e.Timestamp }},
		{"log", func(e *logging.LogEntry) string { return e.TextPayload }},
	}
	filter := fmt.Sprintf(
		"logName:\"run.googleapis.com\" AND resource.type=\"cloud_run_revision\" AND resource.labels.location=\"%v\" AND resource.labels.service_name=\"%v\"",
		region,
		serviceName,
	)
	clf, err := newCloudLogFile(
		ctx,
		apiService.client,
		fields,
		apiService.projectID,
		filter,
	)
	if err != nil {
		return nil, err
	}
	return &cloudRunServiceLog{
		EntryBase:    plugin.NewEntry("log"),
		cloudLogFile: clf,
	}, nil
}

func (crsl *cloudRunServiceLog) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(crsl, "log").
		IsSingleton().
		SetDescription(cloudRunServiceLogDescription)
}

const cloudRunServiceLogDescription = `
This is a cloud run service's log. Each line is formatted as
    LEVEL INSTANCE_ID TIME_UTC LOG
`
