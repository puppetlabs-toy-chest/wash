package datastore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"
	"strings"

	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/log"
)

// Marshalable is an object that can be marshaled and unmarshaled.
type Marshalable interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// CachedMarshalable retrieves a cached item that can be marshaled and unmarshaled.
func CachedMarshalable(cache *bigcache.BigCache, key string, obj Marshalable, cb func() (Marshalable, error)) error {
	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		err = obj.Unmarshal(entry)
		return err
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)
	obj, err = cb()
	if err != nil {
		return err
	}

	entry, err = obj.Marshal()
	if err != nil {
		return err
	}
	cache.Set(key, entry)
	return nil
}

// CachedJSON retrieves cached JSON. If uncached, uses the callback to initialize the cache.
func CachedJSON(cache *bigcache.BigCache, key string, cb func() ([]byte, error)) ([]byte, error) {
	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		return entry, nil
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)
	entry, err = cb()
	if err != nil {
		return nil, err
	}
	cache.Set(key, entry)
	return entry, nil
}

// CachedStrings retrieves a cached array of strings. If uncached, uses the callback to initialize the cache.
// Returned array will always be sorted lexicographically.
func CachedStrings(cache *bigcache.BigCache, key string, cb func() ([]string, error)) ([]string, error) {
	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		var strings []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&strings)
		return strings, err
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Printf("Cache miss on %v", key)
	strings, err := cb()
	if err != nil {
		return nil, err
	}
	sort.Strings(strings)

	if err := CacheAny(cache, key, strings); err != nil {
		return nil, err
	}
	return strings, nil
}

// CacheAny encodes any data as a byte array and stores it in the cache.
func CacheAny(cache *bigcache.BigCache, key string, obj interface{}) error {
	// Guarantee results are sorted.
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(obj); err != nil {
		return err
	}
	cache.Set(key, data.Bytes())
	return nil
}
