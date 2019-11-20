package plugin

import (
	"context"
	"io"

	"github.com/puppetlabs/wash/activity"
)

// ListWithAnalytics is a wrapper to plugin.List. Use it when you need to report
// a 'List' invocation to analytics. Otherwise, use plugin.List
func ListWithAnalytics(ctx context.Context, p Parent) (*EntryMap, error) {
	submitMethodInvocation(ctx, p, "List")
	return List(ctx, p)
}

// OpenWithAnalytics is a wrapper to plugin.Open. Use it when you need to report
// a 'Read' invocation to analytics. Otherwise, use plugin.Open.
func OpenWithAnalytics(ctx context.Context, r Readable) (SizedReader, error) {
	submitMethodInvocation(ctx, r, "Read")
	return Open(ctx, r)
}

// StreamWithAnalytics is a wrapper to s#Stream. Use it when you need to report a 'Stream'
// invocation to analytics. Otherwise, use s#Stream.
func StreamWithAnalytics(ctx context.Context, s Streamable) (io.ReadCloser, error) {
	submitMethodInvocation(ctx, s, "Stream")
	return Stream(ctx, s)
}

// WriteWithAnalytics is a wrapper to w#Write. Use it when you need to report an 'Write'
// invocation to analytics. Otherwise, use w#Write.
func WriteWithAnalytics(ctx context.Context, w Writable, off int64, b []byte) (int, error) {
	submitMethodInvocation(ctx, w, "Write")
	return Write(ctx, w, off, b)
}

// ExecWithAnalytics is a wrapper to e#Exec. Use it when you need to report an 'Exec'
// invocation to analytics. Otherwise, use e#Exec.
func ExecWithAnalytics(ctx context.Context, e Execable, cmd string, args []string, opts ExecOptions) (ExecCommand, error) {
	submitMethodInvocation(ctx, e, "Exec")
	return Exec(ctx, e, cmd, args, opts)
}

// SignalWithAnalytics is a wrapper to plugin.Signal. Use it when you need to report a
// 'Signal' invocation to analytics. Otherwise, use plugin.Signal.
func SignalWithAnalytics(ctx context.Context, s Signalable, signal string) error {
	submitMethodInvocation(ctx, s, "Signal")
	return Signal(ctx, s, signal)
}

// DeleteWithAnalytics is a wrapper to plugin.Delete. Use it when you need to report a
// 'Delete' invocation to analytics. Otherwise, use plugin.Delete.
func DeleteWithAnalytics(ctx context.Context, d Deletable) (bool, error) {
	submitMethodInvocation(ctx, d, "Delete")
	return Delete(ctx, d)
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
