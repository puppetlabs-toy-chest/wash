package datastore

import (
	"time"

	"github.com/hashicorp/vault/helper/locksutil"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

// MemCache is an in-memory cache. It supports concurrent get/set, as well as the ability
// to get-or-update cached data in a single transaction to avoid redundant update activity.
type MemCache struct {
	instance *cache.Cache
	locks    []*locksutil.LockEntry
}

// NewMemCache creates a new MemCache object
func NewMemCache() *MemCache {
	// The TTLs will be passed-in individually in the GetOrUpdate
	// method so we don't need to specify a default expiration
	cache := cache.New(cache.NoExpiration, 3*time.Minute)
	return &MemCache{
		instance: cache,
		locks:    locksutil.CreateLocks(),
	}
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
		return value, nil
	}

	// Cache misses should be rarer, so print them as debug messages.
	log.Debugf("Cache miss on %v", key)
	value, err := generateValue()
	if err != nil {
		return nil, err
	}

	cache.instance.Set(key, value, ttl)

	return value, nil
}
