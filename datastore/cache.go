// Package datastore implements structured data storage for wash server functionality.
package datastore

import (
	"math"
	"regexp"
	"sync"
	"time"

	// TODO: Once https://github.com/patrickmn/go-cache/pull/75
	// is merged, go back to importing the main go-cache repo.
	cache "github.com/ekinanp/go-cache"
	"github.com/hashicorp/vault/helper/locksutil"
	log "github.com/sirupsen/logrus"
)

// Cache is an interface for a cache.
type Cache interface {
	GetOrUpdate(category, key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error)
	Get(category, key string) (interface{}, error)
	Flush()
	Delete(matcher *regexp.Regexp) []string
}

// MemCache is an in-memory cache. It supports concurrent get/set, as well as the ability
// to get-or-update cached data in a single transaction to avoid redundant update activity.
type MemCache struct {
	// Use a write lock when deleting entries to avoid concurrent map read/write on the underlying
	// map used by go-cache. This happened sometimes when evicting an entry at the same time that
	// it's being used again. The scenario became more common when we started evicting cache items
	// when it reaches a limit because lots of new ones are being created over a short period.
	mux         sync.RWMutex
	instance    *cache.Cache
	locks       sync.Map
	hasEviction bool
	limit       int
}

var _ = Cache(&MemCache{})

// NewMemCache creates a new MemCache object
func NewMemCache() *MemCache {
	// The TTLs will be passed-in individually in the GetOrUpdate
	// method so we don't need to specify a default expiration
	cache := cache.New(cache.NoExpiration, 1*time.Minute)
	return &MemCache{
		instance:    cache,
		hasEviction: false,
	}
}

// LockForKey retrieve the lock used for a specific category/key pair.
func (cache *MemCache) lockForKey(category, key string) *locksutil.LockEntry {
	// If a lockset is present for the category, use it. Otherwise create one and add it.
	obj, ok := cache.locks.Load(category)
	if !ok {
		obj, _ = cache.locks.LoadOrStore(category, locksutil.CreateLocks())
	}
	return locksutil.LockForKey(obj.([]*locksutil.LockEntry), key)
}

// WithEvicted adds an eviction function that's called on each object as it's evicted to facilitate
// cleanup.
func (cache *MemCache) WithEvicted(f func(string, interface{})) *MemCache {
	cache.instance.OnEvicted(f)
	cache.hasEviction = true
	return cache
}

// Limit configures a limit to how many entries to keep in the cache. Adding a new one
// evicts the entry closest to expiration.
func (cache *MemCache) Limit(n int) *MemCache {
	cache.limit = n
	return cache
}

func formKey(category, key string) string {
	return category + "::" + key
}

// Get retrieves the value stored at the given key. If not cached, returns (nil, nil).
// Even if a nil value is cached, that's unlikely to be a useful value so we don't see
// a reason to differentiate between absent and nil.
func (cache *MemCache) Get(category, key string) (interface{}, error) {
	key = formKey(category, key)
	value, found := cache.instance.Get(key)
	if found {
		if err, ok := value.(error); ok {
			return nil, err
		}
		return value, nil
	}
	return nil, nil
}

// GetOrUpdate attempts to retrieve the value stored at the given key.
// If the value does not exist, then it generates the value using
// the generateValue function and stores it with the specified ttl.
// If resetTTLOnHit is true, will reset the cache expiration for the entry.
// A ttl of -1 means the item never expires, and a ttl of 0 uses the cache
// default of 1 minute.
func (cache *MemCache) GetOrUpdate(category, key string, ttl time.Duration, resetTTLOnHit bool, generateValue func() (interface{}, error)) (interface{}, error) {
	cache.mux.RLock()
	defer cache.mux.RUnlock()

	l := cache.lockForKey(category, key)
	l.Lock()
	defer l.Unlock()

	// From here on key is a composition of category and key so we can maintain
	// a single cache.
	key = formKey(category, key)
	value, found := cache.instance.Get(key)
	if found {
		log.Tracef("Cache hit on %v", key)
		if resetTTLOnHit {
			// Update last-access time
			cache.instance.Set(key, value, ttl)
		}
		if err, ok := value.(error); ok {
			return nil, err
		}
		return value, nil
	}

	// Cache misses should be rarer, so print them as debug messages.
	log.Debugf("Cache miss on %v", key)

	if cache.limit > 0 && cache.instance.ItemCount() >= cache.limit {
		// Retain write lock when deleting items to avoid concurrent map read/write.
		cache.mux.RUnlock()
		cache.mux.Lock()
		cache.deleteClosestToExpiration()
		cache.mux.Unlock()
		cache.mux.RLock()
	}

	value, err := generateValue()
	// Cache error responses as well. These are often authentication or availability failures
	// and we don't want to continually query the API on failures.
	if err != nil {
		cache.instance.Set(key, err, ttl)
		return nil, err
	}

	cache.instance.Set(key, value, ttl)
	return value, nil
}

func (cache *MemCache) deleteClosestToExpiration() {
	var candidate string
	now := time.Now().UnixNano()
	lowest := int64(math.MaxInt64)
	for k, it := range cache.instance.Items() {
		remaining := it.Expiration - now
		if remaining < lowest {
			lowest = remaining
			candidate = k
		}
	}
	if candidate == "" {
		panic("should have found a candidate")
	}
	cache.instance.Delete(candidate)
}

// Flush deletes all items from the cache. Also resets cache capacity.
// This operation is significantly slower when cache was configured WithEvicted.
func (cache *MemCache) Flush() {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	if cache.hasEviction {
		// Flush doesn't trigger the eviction callback. If we've registered one, ensure it's
		// triggered for all keys being removed. First delete all valid entries, then delete
		// expired entries (the reverse would be incorrect, as entries might expire after
		// calling DeleteExpired but before calling Items).
		for k := range cache.instance.Items() {
			cache.instance.Delete(k)
		}
		cache.instance.DeleteExpired()
	}
	cache.instance.Flush()
}

// Delete removes entries from the cache that match the provided regexp.
func (cache *MemCache) Delete(matcher *regexp.Regexp) []string {
	cache.mux.Lock()
	defer cache.mux.Unlock()

	log.Debugf("Deleting matches for %v", matcher)
	items := cache.instance.Items()
	deleted := make([]string, 0, len(items))
	for k := range items {
		if matcher.MatchString(k) {
			log.Debugf("Deleting cache entry %v", k)
			cache.instance.Delete(k)
			deleted = append(deleted, k)
		} else {
			log.Debugf("Skipping %v", k)
		}
	}
	return deleted
}
