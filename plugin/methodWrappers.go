package plugin

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/puppetlabs/wash/activity"
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

// Attributes returns the entry's attributes.
func Attributes(e Entry) EntryAttributes {
	return e.attributes()
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

// Read reads up to size bits of the entry's content starting at the given offset.
// It will panic if the entry does not support the read action. Callers can use
// len(data) to check the amount of data that was actually read.
//
// If the offset is >= to the available content size, then data != nil, len(data) == 0,
// and err == io.EOF. Otherwise if len(data) < size, then err == io.EOF.
//
// Note that Read is thread-safe.
func Read(ctx context.Context, e Entry, size int64, offset int64) (data []byte, err error) {
	if !ReadAction().IsSupportedOn(e) {
		panic("plugin.Read called on a non-readable entry")
	}
	if size < 0 {
		return nil, fmt.Errorf("called with a negative size %v", size)
	}
	if offset < 0 {
		return nil, fmt.Errorf("called with a negative offset %v", offset)
	}
	data = []byte{}
	inputSize := size
	if attr := e.attributes(); attr.HasSize() {
		contentSize := int64(attr.Size())
		if offset >= contentSize {
			err = io.EOF
			return
		}
		minSize := size + offset
		if contentSize < minSize {
			size = contentSize
			err = io.EOF
		}
	} else if ReadAction().signature(e) == blockReadableSignature {
		activity.Warnf(ctx, "size attribute not set for block-readable entry %v", e.id())
	}
	content, contentErr := cachedRead(ctx, e)
	if contentErr != nil {
		err = contentErr
		return
	}
	data, readErr := content.read(ctx, size, offset)
	if readErr != nil {
		err = readErr
	}
	if actualSize := int64(len(data)); actualSize > size {
		return nil, fmt.Errorf("requested %v bytes (input was %v), but plugin's API returned %v bytes", size, inputSize, actualSize)
	}
	return
}

// Size returns the size of readable content (if we can determine it).
func Size(ctx context.Context, e Entry) (uint64, error) {
	if !ReadAction().IsSupportedOn(e) {
		panic("plugin.Read called on a non-readable entry")
	}

	data, err := cachedRead(ctx, e)
	if err != nil {
		return 0, err
	}
	return data.size(), nil
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

// Write sends the supplied buffer to the entry.
func Write(ctx context.Context, a Writable, b []byte) error {
	return a.Write(ctx, b)
}

// Signal signals the entry with the specified signal
func Signal(ctx context.Context, s Signalable, signal string) error {
	// Signals are case-insensitive
	signal = strings.ToLower(signal)

	// Validate the provided signal if the entry's schema is available
	schema, err := Schema(s)
	if err != nil {
		return fmt.Errorf("failed to retrieve the entry's schema for signal validation: %w", err)
	}
	if schema != nil {
		var validSignals []string
		var validSignalGroups []string
		var isValidSignal bool
		for _, signalSchema := range schema.Signals {
			if signalSchema.IsGroup() {
				validSignalGroups = append(validSignalGroups, signalSchema.Name())
				isValidSignal = signalSchema.Regex().MatchString(signal)
			} else {
				validSignals = append(validSignals, signalSchema.Name())
				isValidSignal = signalSchema.Name() == signal
			}
			if isValidSignal {
				break
			}
		}
		if !isValidSignal {
			errMsg := fmt.Sprintf("invalid signal %v", signal)
			if len(validSignals) > 0 {
				errMsg += fmt.Sprintf(". Valid signals are %v", strings.Join(validSignals, ", "))
			}
			if len(validSignalGroups) > 0 {
				errMsg += fmt.Sprintf(". Valid signal groups are %v", strings.Join(validSignalGroups, ", "))
			}
			return InvalidInputErr{errMsg}
		}
	}

	// Go ahead and send the signal
	err = s.Signal(ctx, signal)
	if err != nil {
		return err
	}

	// The signal was successfully sent. Clear the entry's cache and its parent's
	// cached list result to ensure that fresh data's loaded when needed
	ClearCacheFor(s.id(), true)
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
	//
	// If !deleted, the entry will eventually be deleted. However it's likely that
	// Delete did update the entry (e.g. on VMs, Delete causes a state transition).
	// Thus we also clear the parent's cached list result to ensure fresh data.
	ClearCacheFor(d.id(), !deleted)
	if deleted {
		// The entry was deleted, so delete the entry from the parent's cached list
		// result.
		parentID, cname := splitID(d.id())
		listOpName := defaultOpCodeToNameMap[ListOp]
		entries, _ := cache.Get(listOpName, parentID)
		if entries != nil {
			entries.(*EntryMap).Delete(cname)
		}
	}

	return
}
