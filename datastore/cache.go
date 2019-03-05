package datastore

import (
	"time"

	"github.com/hashicorp/vault/helper/locksutil"
	cache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

// MemCache is an in-memory cache. It supports concurrent get/set, as well as the ability
// to get-or-update cached data in a single transaction to avoid redundant update activity.
type MemCache struct {
	instance    *cache.Cache
	locks       []*locksutil.LockEntry
	hasEviction bool
}

// NewMemCache creates a new MemCache object
func NewMemCache() *MemCache {
	// The TTLs will be passed-in individually in the GetOrUpdate
	// method so we don't need to specify a default expiration
	cache := cache.New(cache.NoExpiration, 1*time.Minute)
	return &MemCache{
		instance:    cache,
		locks:       locksutil.CreateLocks(),
		hasEviction: false,
	}
}

// NewMemCacheWithEvicted creates a new MemCache object that calls the provided eviction function
// on each object as it's evicted to facilitate cleanup.
func NewMemCacheWithEvicted(f func(string, interface{})) *MemCache {
	cache := NewMemCache()
	cache.instance.OnEvicted(f)
	cache.hasEviction = true
	return cache
}

// LockForKey retrieve the lock used for a specific key.
func (cache *MemCache) lockForKey(key string) *locksutil.LockEntry {
	return locksutil.LockForKey(cache.locks, key)
}

// GetOrUpdate attempts to retrieve the value stored at the given key.
// If the value does not exist, then it generates the value using
// the generateValue function and stores it with the specified ttl.
func (cache *MemCache) GetOrUpdate(key string, ttl time.Duration, generateValue func() (interface{}, error)) (interface{}, error) {
	l := cache.lockForKey(key)
	l.Lock()
	defer l.Unlock()

	value, found := cache.instance.Get(key)
	if found {
		log.Tracef("Cache hit on %v", key)
		if err, ok := value.(error); ok {
			return nil, err
		}
		return value, nil
	}

	// Cache misses should be rarer, so print them as debug messages.
	log.Debugf("Cache miss on %v", key)
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

// Flush deletes all items from the cache.
// This operation is significantly slower when cache was created with NewMemCacheWithEvicted.
func (cache *MemCache) Flush() {
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
