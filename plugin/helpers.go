package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

// NewEntry creates a new named entry
func NewEntry(name string) EntryBase {
	return EntryBase{name}
}

// JSONToJSONMap converts JSON to a map of its top-level keys to JSON serialized values.
func JSONToJSONMap(inrec []byte) (map[string][]byte, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(inrec, &data); err != nil {
		return nil, err
	}

	var err error
	d := make(map[string][]byte)
	for k, v := range data {
		d[k], err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}
	return d, nil
}

// TrackTime helper is useful for timing functions.
// Use with `defer plugin.TrackTime(time.Now(), "funcname")`.
func TrackTime(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Infof("%s took %s", name, elapsed)
}

// PrefetchOpen can be called to open a file for DefaultTimeout (if it supports Close).
// Commonly used as `go PrefetchOpen(...)` to kick off prefetching asynchronously.
func PrefetchOpen(file Readable) {
	buf, err := file.Open(context.Background())
	if closer, ok := buf.(io.Closer); err == nil && ok {
		go func() {
			time.Sleep(DefaultTimeout)
			closer.Close()
		}()
	}
}

// FindEntryByName finds an entry by name within the given group
func FindEntryByName(ctx context.Context, group Group, name string) (Entry, error) {
	entries, err := group.LS(ctx)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.Name() == name {
			return entry, nil
		}
	}

	return nil, fmt.Errorf("Could not find entry %v in group %v", name, group.Name())
}

// FindEntryByPath finds an entry in the group from a given path
func FindEntryByPath(ctx context.Context, group Group, segments []string) (Entry, error) {
	var curEntry Entry
	curEntry = group

	for _, segment := range segments {
		switch group := curEntry.(type) {
		case Group:
			entry, err := FindEntryByName(ctx, group, segment)
			if err != nil {
				return nil, err
			}

			curEntry = entry
		default:
			// TODO: Make this return a structured error. This would let us distinguish
			// between different cases (e.g. Not Found vs. IO error vs. Malformed path)
			return nil, fmt.Errorf("Segment %v of path %v is not a Group", curEntry.Name(), strings.Join(segments, "/"))
		}
	}

	return curEntry, nil
}
