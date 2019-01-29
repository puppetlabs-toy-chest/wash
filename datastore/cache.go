package datastore

import (
	"github.com/allegro/bigcache"
	"github.com/puppetlabs/wash/log"
)

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
