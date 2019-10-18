package plugin

import (
	"context"
	"io"

	"github.com/puppetlabs/wash/activity"
)

// ListWithAnalytics is a wrapper to plugin.CachedList. Use it when you need to report
// a 'List' invocation to analytics. Otherwise, use plugin.CachedList
func ListWithAnalytics(ctx context.Context, p Parent) (map[string]Entry, error) {
	submitMethodInvocation(ctx, p, "List")
	return CachedList(ctx, p)
}

// OpenWithAnalytics is a wrapper to plugin.CachedOpen. Use it when you need to report
// a 'Read' invocation to analytics. Otherwise, use plugin.CachedOpen
func OpenWithAnalytics(ctx context.Context, r Readable) (SizedReader, error) {
	submitMethodInvocation(ctx, r, "Read")
	return CachedOpen(ctx, r)
}

// StreamWithAnalytics is a wrapper to s#Stream. Use it when you need to report a 'Stream'
// invocation to analytics. Otherwise, use s#Stream
func StreamWithAnalytics(ctx context.Context, s Streamable) (io.ReadCloser, error) {
	submitMethodInvocation(ctx, s, "Stream")
	return s.Stream(ctx)
}

// ExecWithAnalytics is a wrapper to e#Exec. Use it when you need to report an 'Exec'
// invocation to analytics. Otherwise, use e#Exec.
func ExecWithAnalytics(ctx context.Context, e Execable, cmd string, args []string, opts ExecOptions) (ExecCommand, error) {
	submitMethodInvocation(ctx, e, "Exec")
	return e.Exec(ctx, cmd, args, opts)
}

func submitMethodInvocation(ctx context.Context, e Entry, method string) {
	isCorePluginEntry := e.Schema() != nil
	if !isCorePluginEntry {
		return
	}
	plugin := pluginName(e)
	// Asynchronously submit the method invocation so that we do not wait
	// on a Flush operation
	go activity.SubmitMethodInvocation(
		ctx,
		plugin,
		TypeID(e),
		method,
	)
}
