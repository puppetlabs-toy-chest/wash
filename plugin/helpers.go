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

// ToMetadata converts an object to a metadata result. If the input is already an array of bytes, it
// must contain a serialized JSON object. Will panic if given something besides a struct or []byte.
func ToMetadata(obj interface{}) map[string]interface{} {
	var err error
	var inrec []byte
	if arr, ok := obj.([]byte); ok {
		inrec = arr
	} else {
		if inrec, err = json.Marshal(obj); err != nil {
			// Internal error if we can't marshal an object
			panic(err)
		}
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(inrec, &meta); err != nil {
		// Internal error if not a JSON object
		panic(err)
	}
	return meta
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

// ErrEntryDoesNotExist is an error indicating that the entry specified
// by the given path does not exist in the given group
type ErrEntryDoesNotExist struct {
	group Group
	path  string
}

func (e ErrEntryDoesNotExist) Error() string {
	return fmt.Sprintf("The %v entry does not exist in the %v group", e.path, e.group.Name())
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

	return nil, ErrEntryDoesNotExist{group, name}
}

// ErrInvalidEntryPath is an error indicating an invalid entry path
type ErrInvalidEntryPath struct {
	path   string
	reason string
}

func (e ErrInvalidEntryPath) Error() string {
	return fmt.Sprintf("%v is an invalid entry path: %v", e.path, e.reason)
}

// FindEntryByPath finds an entry in the group from a given path
func FindEntryByPath(ctx context.Context, startGroup Group, segments []string) (Entry, error) {
	var curEntry Entry
	curEntry = startGroup

	visitedSegments := make([]string, 0, cap(segments))
	for _, segment := range segments {
		switch curGroup := curEntry.(type) {
		case Group:
			entry, err := FindEntryByName(ctx, curGroup, segment)
			visitedSegments = append(visitedSegments, segment)

			if err != nil {
				if _, ok := err.(ErrEntryDoesNotExist); ok {
					err = ErrEntryDoesNotExist{startGroup, strings.Join(visitedSegments, "/")}
				}

				return nil, err
			}

			curEntry = entry
		default:
			reason := fmt.Sprintf("The entry %v is not a group", strings.Join(visitedSegments, "/"))
			return nil, ErrInvalidEntryPath{strings.Join(segments, "/"), reason}
		}
	}

	return curEntry, nil
}
