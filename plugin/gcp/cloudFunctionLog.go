package gcp

import (
	"context"
	"fmt"

	"github.com/puppetlabs/wash/plugin"
	"google.golang.org/api/logging/v2"
)

type cloudFunctionLog struct {
	plugin.EntryBase
	*cloudLogFile
}

func newCloudFunctionLog(ctx context.Context, service cloudFunctionsProjectService, region string, functionName string) (*cloudFunctionLog, error) {
	fields := []cloudLogEntryField{
		// Note that we leave the severity and timestamp as-is to make them easier
		// to parse with existing log libraries. This is slightly different from
		// `gcloud function logs read <function_name>`
		severityField,
		{"execution_id", func(e *logging.LogEntry) string { return e.Labels["execution_id"] }},
		timestampField,
		msgField,
	}
	filter := fmt.Sprintf(
		"logName:\"cloud-functions\" AND resource.type=\"cloud_function\" AND resource.labels.region=\"%v\" AND resource.labels.function_name=\"%v\"",
		region,
		functionName,
	)
	clf, err := newCloudLogFile(
		ctx,
		service.client,
		fields,
		service.projectID,
		filter,
	)
	if err != nil {
		return nil, err
	}
	return &cloudFunctionLog{
		EntryBase:    plugin.NewEntry("log"),
		cloudLogFile: clf,
	}, nil
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
