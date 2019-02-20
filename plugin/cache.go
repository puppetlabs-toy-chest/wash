package plugin

import (
	"context"
	"time"

	"github.com/puppetlabs/wash/datastore"
)

// TODO: If space becomes an issue, we can make this an
// int8, and CacheConfig#ttl a []time.Duration so that
// the ops can be used to index into the array.
type cachedOp string

const (
	// LS represents the string name of Group#LS
	LS cachedOp = "LS"
	// Open represents the string name of Readable#Open
	Open cachedOp = "Open"
	// Metadata represents the string name of Metadata#Open
	Metadata cachedOp = "Metadata"
)

var allCachedOps = []cachedOp{
	LS,
	Open,
	Metadata,
}

// CacheConfig represents an entry's cache configuration
type CacheConfig struct {
	ttl map[cachedOp]time.Duration
}

func newCacheConfig() *CacheConfig {
	config := &CacheConfig{}
	config.ttl = make(map[cachedOp]time.Duration)

	for _, op := range allCachedOps {
		config.SetTTLOf(op, 5*time.Second)
	}

	return config
}

func (config *CacheConfig) getTTLOf(op cachedOp) time.Duration {
	return config.ttl[op]
}

// SetTTLOf sets the specified op's TTL
func (config *CacheConfig) SetTTLOf(op cachedOp, ttl time.Duration) {
	config.ttl[op] = ttl
}

// TurnOffCachingFor turns off caching for the specified op
func (config *CacheConfig) TurnOffCachingFor(op cachedOp) {
	config.SetTTLOf(op, -1)
}

// TurnOffCaching turns off caching for all ops
func (config *CacheConfig) TurnOffCaching() {
	for _, op := range allCachedOps {
		config.TurnOffCachingFor(op)
	}
}

var cache *datastore.MemCache

// InitCache initializes the cache
func InitCache() {
	cache = datastore.NewMemCache()
}

func cachedOpHelper(op cachedOp, entry Entry, id string, generateValue func() (interface{}, error)) (interface{}, error) {
	if cache == nil {
		panic("The cache was not initialized. You can initialize the cache by invoking plugin.InitCache()")
	}

	ttl := entry.CacheConfig().getTTLOf(op)
	if ttl < 0 {
		return generateValue()
	}

	return cache.GetOrUpdate(string(op)+"::"+id, ttl, generateValue)
}

// CachedLS caches a Group object's LS method
func CachedLS(g Group, id string, ctx context.Context) ([]Entry, error) {
	cachedEntries, err := cachedOpHelper(LS, g, id, func() (interface{}, error) {
		return g.LS(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedEntries.([]Entry), nil
}

// CachedOpen caches a Readable object's Open method
func CachedOpen(r Readable, id string, ctx context.Context) (SizedReader, error) {
	cachedContent, err := cachedOpHelper(Open, r, id, func() (interface{}, error) {
		return r.Open(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedContent.(SizedReader), nil
}

// CachedMetadata caches a Resource object's Metadata method
func CachedMetadata(r Resource, id string, ctx context.Context) (map[string]interface{}, error) {
	cachedMetadata, err := cachedOpHelper(Metadata, r, id, func() (interface{}, error) {
		return r.Metadata(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedMetadata.(map[string]interface{}), nil
}
