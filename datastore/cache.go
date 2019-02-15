package datastore

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"sort"
	"time"

	"github.com/allegro/bigcache"
	"github.com/hashicorp/vault/helper/locksutil"
	log "github.com/sirupsen/logrus"
)

// TTL is a selection of cache TTLs supported by the library: Slow and Fast.
type TTL time.Duration

// Selection of cache TTLs supported by the library.
// Fast: used for HTTP requests.
// Slow: used for operations that require spinning up a container.
const (
	Fast = TTL(5 * time.Second)
	Slow = TTL(60 * time.Minute)
)

// Backends lists available backend keys.
var Backends = []TTL{Slow, Fast}

// MemCache is an in-memory cache. It supports concurrent get/set, as well as the ability
// to get-or-update cached data in a single transaction to avoid redundant update activity.
type MemCache struct {
	backends map[TTL]*bigcache.BigCache
	locks    []*locksutil.LockEntry
}

// NewMemCache creates a new in-memory cache populated with available TTLs.
func NewMemCache() (*MemCache, error) {
	backends := make(map[TTL]*bigcache.BigCache)
	for _, ttl := range Backends {
		config := bigcache.DefaultConfig(time.Duration(ttl))
		config.CleanWindow = time.Duration(ttl)
		backend, err := bigcache.NewBigCache(config)
		if err != nil {
			return nil, err
		}
		backends[ttl] = backend
	}
	return &MemCache{backends, locksutil.CreateLocks()}, nil
}

// Get a cached entry by key from the cache.
func (cache *MemCache) Get(key string) ([]byte, error) {
	chans := make([]chan []byte, len(Backends))
	for i, ttl := range Backends {
		chans[i] = make(chan []byte, 1)
		go func(t TTL, ch chan []byte) {
			val, err := cache.backends[t].Get(key)
			if err == nil {
				ch <- val
			}
			close(ch)
		}(ttl, chans[i])
	}

	// TODO: https://stackoverflow.com/a/19992525/2048059 ?
	ch0, ch1 := chans[0], chans[1]
	for ch0 != nil || ch1 != nil {
		select {
		case x, ok := <-ch0:
			if ok {
				return x, nil
			}
			ch0 = nil
		case x, ok := <-ch1:
			if ok {
				return x, nil
			}
			ch1 = nil
		}
	}
	return nil, fmt.Errorf("Entry %q not found", key)
}

// Set caches the entry by key with a fast TTL.
func (cache *MemCache) Set(key string, entry []byte) error {
	return cache.backends[Fast].Set(key, entry)
}

// SetSlow caches the entry by key with a slow TTL.
func (cache *MemCache) SetSlow(key string, entry []byte) error {
	return cache.backends[Slow].Set(key, entry)
}

// SetAny caches any object by key with the specified TTL using the gob encoder.
func (cache *MemCache) SetAny(key string, obj interface{}, ttl TTL) error {
	backend, ok := cache.backends[ttl]
	if !ok {
		return fmt.Errorf("Unknown TTL %q requested", ttl)
	}
	var data bytes.Buffer
	enc := gob.NewEncoder(&data)
	if err := enc.Encode(obj); err != nil {
		return err
	}
	backend.Set(key, data.Bytes())
	return nil
}

// LockForKey retrieve the lock used for a specific key.
func (cache *MemCache) LockForKey(key string) *locksutil.LockEntry {
	return locksutil.LockForKey(cache.locks, key)
}

// LocksForKeys retrieves a list of locks used for a specific key. They must be locked
// in-order to avoid deadlocks.
func (cache *MemCache) LocksForKeys(keys []string) []*locksutil.LockEntry {
	return locksutil.LocksForKeys(cache.locks, keys)
}

// Marshalable is an object that can be marshaled and unmarshaled.
type Marshalable interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// CachedMarshalable retrieves a cached item that can be marshaled and unmarshaled.
func (cache *MemCache) CachedMarshalable(key string, obj Marshalable, cb func() (Marshalable, error)) error {
	// Acquire a lock for the duration of this operation.
	l := cache.LockForKey(key)
	l.Lock()
	defer l.Unlock()

	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		err = obj.Unmarshal(entry)
		return err
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Infof("Cache miss on %v", key)
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
func (cache *MemCache) CachedJSON(key string, cb func() ([]byte, error)) ([]byte, error) {
	l := cache.LockForKey(key)
	l.Lock()
	defer l.Unlock()

	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		return entry, nil
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Infof("Cache miss on %v", key)
	entry, err = cb()
	if err != nil {
		return nil, err
	}
	cache.Set(key, entry)
	return entry, nil
}

// CachedStrings retrieves a cached array of strings. If uncached, uses the callback to initialize the cache.
// Returned array will always be sorted lexicographically.
func (cache *MemCache) CachedStrings(key string, cb func() ([]string, error)) ([]string, error) {
	l := cache.LockForKey(key)
	l.Lock()
	defer l.Unlock()

	entry, err := cache.Get(key)
	if err == nil {
		log.Debugf("Cache hit on %v", key)
		var strings []string
		dec := gob.NewDecoder(bytes.NewReader(entry))
		err = dec.Decode(&strings)
		return strings, err
	}

	// Cache misses should be rarer, so always print them. Frequent messages are a sign of problems.
	log.Infof("Cache miss on %v", key)
	strings, err := cb()
	if err != nil {
		return nil, err
	}
	sort.Strings(strings)

	if err := cache.SetAny(key, strings, Fast); err != nil {
		return nil, err
	}
	return strings, nil
}
