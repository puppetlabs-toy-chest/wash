package plugin

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// InvalidInputErr indicates that the method invocation received invalid
// input (e.g. plugin.Signal received an unsupported signal).
type InvalidInputErr struct {
	reason string
}

func (e InvalidInputErr) Error() string {
	return e.reason
}

// IsInvalidInputErr returns true if err is an InvalidInputErr error object
func IsInvalidInputErr(err error) bool {
	_, ok := err.(InvalidInputErr)
	return ok
}

// This file contains all of the plugin.<Method> wrappers. You should
// invoke plugin.<Method> instead of e.<Method> because plugin.<Method>
// could contain additional, plugin-agnostic code required to get e.<Method>
// working correctly (like e.g. caching and validation).

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

/*
Name returns the entry's name as it was passed into
plugin.NewEntry. It is meant to be called by other
Wash packages. Plugin authors should use EntryBase#Name
when writing their plugins.
*/
func Name(e Entry) string {
	// The reason we don't expose EntryBase#Name in the Entry
	// interface is so plugin authors don't override it. It ensures
	// that whatever name they pass into plugin.NewEntry is the
	// name received by Wash.
	return e.name()
}

/*
CName returns the entry's canonical name, which is what Wash uses to
construct the entry's path. The entry's cname is plugin.Name(e), but with
all '/' characters replaced by a '#' character. CNames are necessary
because it is possible for entry names to have '/'es in them, which is
illegal in bourne shells and UNIX-y filesystems.

CNames are unique. CName uniqueness is checked in plugin.CachedList.

NOTE: The '#' character was chosen because it is unlikely to appear in
a meaningful entry's name. If, however, there's a good chance that an
entry's name can contain the '#' character, and that two entries can
have the same cname (e.g. 'foo/bar', 'foo#bar'), then you can use
e.SetSlashReplacer(<char>) to change the default slash replacer from
a '#' to <char>.
*/
func CName(e Entry) string {
	if len(e.name()) == 0 {
		panic("plugin.CName: e.name() is empty")
	}
	// We make the CName a separate function instead of embedding it
	// in the Entry interface because doing so prevents plugin authors
	// from overriding it.
	return strings.Replace(
		e.name(),
		"/",
		string(e.slashReplacer()),
		-1,
	)
}

// ID returns the entry's ID, which is just its path rooted at Wash's mountpoint.
// An entry's ID is described as
//     /<plugin_name>/<parent1_cname>/<parent2_cname>/.../<entry_cname>
//
// NOTE: <plugin_name> is really <plugin_cname>. However since <plugin_name>
// can never contain a '/', <plugin_cname> reduces to <plugin_name>.
func ID(e Entry) string {
	if e.id() == "" {
		msg := fmt.Sprintf("plugin.ID: entry %v (cname %v) has no ID", e.name(), CName(e))
		panic(msg)
	}
	return e.id()
}

// Attributes returns the entry's attributes. If size is unknown, it will check whether the entry
// has locally cached content and if so set that for the size.
func Attributes(e Entry) EntryAttributes {
	// Sometimes an entry doesn't know its size unless it's already downloaded some content. Having
	// to download content makes list slow, and is a burden for external plugin developers. Check if
	// we already know the size. If not, FUSE will use a reasonable default so tools don't ignore it.
	attr := e.attributes()
	if !attr.HasSize() && cache != nil {
		// We have no way to preserve this on the entry, and it likely wouldn't help because we often
		// recreate the entry to ensure we have an accurate representation. So when the cache expires
		// we revert to stating the size is unknown until the next read operation.
		if val, _ := cache.Get(defaultOpCodeToNameMap[OpenOp], e.id()); val != nil {
			rdr := val.(SizedReader)
			attr.SetSize(uint64(rdr.Size()))
		}
	}
	return attr
}

// IsPrefetched returns whether an entry has data that was added during creation that it would
// like to have updated.
func IsPrefetched(e Entry) bool {
	return e.isPrefetched()
}

// Schema returns the entry's schema.
func Schema(e Entry) (*EntrySchema, error) {
	return schema(e)
}

// List lists the parent's children. It returns an EntryMap to optimize querying a specific
// entry.
//
// Note that List's results could be cached.
func List(ctx context.Context, p Parent) (*EntryMap, error) {
	return cachedList(ctx, p)
}

// Open reads the entry's content. Note that Open's results could be cached. Thus, when
// using the reader returned by this method, use idempotent read operations such as ReadAt
// or wrap it in a SectionReader. Using Read operations on the cached reader will change it
// and make subsequent uses of the cached reader invalid.
//
// TODO: Could we change this to Read? E.g. plugin.Read.
func Open(ctx context.Context, r Readable) (SizedReader, error) {
	return cachedOpen(ctx, r)
}

// Metadata returns the entry's metadata. Note that Metadata's results could be cached.
func Metadata(ctx context.Context, e Entry) (JSONObject, error) {
	return cachedMetadata(ctx, e)
}

// Exec execs the command on the given entry.
func Exec(ctx context.Context, e Execable, cmd string, args []string, opts ExecOptions) (ExecCommand, error) {
	return e.Exec(ctx, cmd, args, opts)
}

// Stream streams the entry's content for updates.
func Stream(ctx context.Context, s Streamable) (io.ReadCloser, error) {
	return s.Stream(ctx)
}

// Signal signals the entry with the specified signal
func Signal(ctx context.Context, s Signalable, signal string) error {
	// Signals are case-insensitive
	signal = strings.ToLower(signal)

	// Validate the provided signal if the entry's schema is available
	schema, err := Schema(s)
	if err != nil {
		return fmt.Errorf("failed to retrieve the entry's schema for signal validation: %v", err)
	}
	if schema != nil {
		if _, ok := schema.Signals[signal]; !ok {
			var validSignals []string
			for signal := range schema.Signals {
				validSignals = append(validSignals, signal)
			}
			sort.Strings(validSignals)
			return InvalidInputErr{
				fmt.Sprintf("invalid signal %v. Valid signals are %v", signal, strings.Join(validSignals, ", ")),
			}
		}
	}

	// Go ahead and send the signal
	err = s.Signal(ctx, signal)
	if err != nil {
		return err
	}

	// The signal was successfully sent. Clear the entry's cache and its parent's
	// cached list result to ensure that fresh data's loaded when needed
	ClearCacheFor(s.id())
	parentID, _ := splitID(s.id())
	listOpName := defaultOpCodeToNameMap[ListOp]
	entries, _ := cache.Get(listOpName, parentID)
	if entries != nil {
		cache.Delete(opKeyRegex(listOpName, parentID))
	}

	return nil
}

// Delete deletes the given entry.
func Delete(ctx context.Context, d Deletable) (deleted bool, err error) {
	deleted, err = d.Delete(ctx)
	if err != nil {
		return
	}

	// Delete was successful, so update the cache to ensure that fresh data's loaded
	// when needed. This includes:
	//   * Clearing the entry and its children's cache.
	//   * Updating the parent's cached list result.
	ClearCacheFor(d.id())
	parentID, cname := splitID(d.id())
	listOpName := defaultOpCodeToNameMap[ListOp]
	entries, _ := cache.Get(listOpName, parentID)
	if entries == nil {
		return
	}
	if deleted {
		// The entry was deleted, so delete the entry from the parent's cached list
		// result.
		entries.(*EntryMap).Delete(cname)
	} else {
		// The entry will eventually be deleted. However it's likely that Delete
		// did update the entry (e.g. on VMs, Delete causes a state transition).
		// Thus, we clear the parent's cached list result to ensure fresh data.
		//
		// Note that this is symmetric with Signal. This makes sense because for
		// some entries, deleting that entry is equivalent to sending a termination
		// signal (e.g. EC2 instances).
		cache.Delete(opKeyRegex(listOpName, parentID))
	}

	return
}

// returns (parentID, cname)
func splitID(entryID string) (string, string) {
	segments := strings.Split(entryID, "/")
	parentID := strings.Join(segments[:len(segments)-1], "/")
	cname := segments[len(segments)-1]
	return parentID, cname
}
