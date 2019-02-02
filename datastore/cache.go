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

// ContainsString returns whether the named string is included in strings,
// assuming that the array of strings is sorted.
func ContainsString(names []string, name string) bool {
	idx := sort.SearchStrings(names, name)
	return idx < len(names) && names[idx] == name
}

// FindCompositeString returns whether the name is present in the array of sorted composite
// strings. Composite strings are token1/token2, where name is matched against token1.
func FindCompositeString(names []string, name string) (string, bool) {
	idx := sort.Search(len(names), func(i int) bool {
		x, _ := SplitCompositeString(names[i])
		return x >= name
	})
	if idx < len(names) {
		x, _ := SplitCompositeString(names[idx])
		if x == name {
			return names[idx], true
		}
	}
	return "", false
}

// SplitCompositeString splits a string around a '/' separator, requiring that the string only have
// a single separator.
func SplitCompositeString(id string) (string, string) {
	tokens := strings.Split(id, "/")
	if len(tokens) != 2 {
		panic(fmt.Sprintf("SplitCompositeString given an invalid name/id pair: %v", id))
	}
	return tokens[0], tokens[1]
}

// MakeCompositeString makes a composite string by joining name and extra with a '/' separator.
func MakeCompositeString(name string, extra string) string {
	return name + "/" + extra
}
