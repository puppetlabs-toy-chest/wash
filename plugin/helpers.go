package plugin

import (
	"encoding/json"
	"time"

	"github.com/Benchkram/errz"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// DefaultTimeout is the default timeout for prefetching
var DefaultTimeout = 10 * time.Second

// NewEntry creates a new named entry
func NewEntry(name string) EntryBase {
	return EntryBase{name, newCacheConfig()}
}

// ToMetadata converts an object to a metadata result. If the input is already an array of bytes, it
// must contain a serialized JSON object. Will panic if given something besides a struct or []byte.
func ToMetadata(obj interface{}) MetadataMap {
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
	var meta MetadataMap
	// Internal error if not a JSON object
	errz.Fatal(json.Unmarshal(inrec, &meta))
	return meta
}

// TrackTime helper is useful for timing functions.
// Use with `defer plugin.TrackTime(time.Now(), "funcname")`.
func TrackTime(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Infof("%s took %s", name, elapsed)
}

// LogErr logs an error + context directly to console or file. Uses logrus instead of
// golang's log library.
func LogErr(err error, msgs ...string) {
	if err != nil {
		for _, msg := range msgs {
			err = errors.Wrap(err, msg)
		}
		log.Printf("%+v", err)
	}
}
