package plugin

import (
	"context"
	"flag"
	"regexp"
	"strings"
	"time"

	"github.com/puppetlabs/wash/datastore"
)

type cachedOp int8

const (
	// List represents Group#List
	List cachedOp = iota
	// Open represents Readable#Open
	Open
	// Metadata represents Resource#Metadata
	Metadata
)

var cachedOpToNameMap = [3]string{"List", "Open", "Metadata"}

// CacheConfig represents an entry's cache configuration
type CacheConfig struct {
	ttl [3]time.Duration
}

func newCacheConfig() *CacheConfig {
	config := &CacheConfig{}

	for op := range config.ttl {
		config.SetTTLOf(cachedOp(op), 15*time.Second)
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
	for op := range config.ttl {
		config.TurnOffCachingFor(cachedOp(op))
	}
}

var cache datastore.Cache
var defaultConfig CacheConfig

func notRunningTests() bool {
	return flag.Lookup("test.v") == nil
}

// InitCache initializes the cache
func InitCache() {
	if notRunningTests() {
		cache = datastore.NewMemCache()
		defaultConfig = *newCacheConfig()
	} else {
		panic("InitCache can only be called in production. Tests should call SetTestCache instead.")
	}
}

// SetTestCache sets the cache to the provided mock. It can only be called
// by the tests
func SetTestCache(c datastore.Cache) {
	if notRunningTests() {
		panic("SetTestCache can only be called when running the tests")
	}

	if cache != nil {
		panic("The test cache has already been set")
	}

	cache = c
}

// UnsetTestCache unsets the test cache. It can only be called
// by the tests
func UnsetTestCache() {
	if notRunningTests() {
		panic("UnsetTestCache can only be called when running the tests")
	}

	if cache == nil {
		panic("The test cache has already been unset")
	}

	cache = nil
}

// This method exists to simplify ClearCacheFor's tests.
// Specifically, it lets us decouple the regex's correctness
// from the cache's implementation.
func opKeysRegex(path string) (*regexp.Regexp, error) {
	opQualifier := "^[a-zA-Z]*::"

	var expr string
	if path == "/" {
		expr = opQualifier + "/.*"
	} else {
		expr = opQualifier + "/" + strings.Trim(path, "/") + "($|/.*)"
	}

	return regexp.Compile(expr)
}

// ClearCacheFor removes entries from the cache that match or are children of the provided path.
// If successful, returns an array of deleted keys.
//
// TODO: If path == "/", we could optimize this by calling cache.Flush(). Not important
// right now, but may be worth considering in the future.
func ClearCacheFor(path string) ([]string, error) {
	rx, err := opKeysRegex(path)
	if err != nil {
		return nil, err
	}

	return cache.Delete(rx), nil
}

func cachedOpHelper(op cachedOp, entry Entry, id string, generateValue func() (interface{}, error)) (interface{}, error) {
	if cache == nil {
		if notRunningTests() {
			panic("The cache was not initialized. You can initialize the cache by invoking plugin.InitCache()")
		} else {
			panic("The test cache was not set. You can set it by invoking plugin.SetTestCache(<cache>)")
		}
	}

	cached, ok := entry.(Cached)
	if !ok {
		return generateValue()
	}

	config := cached.CacheConfig()
	if config == nil {
		config = &defaultConfig
	}

	ttl := config.getTTLOf(op)
	if ttl < 0 {
		return generateValue()
	}

	opName := cachedOpToNameMap[op]
	return cache.GetOrUpdate(opName+"::"+id, ttl, false, generateValue)
}

// CachedList caches a Group object's List method
func CachedList(ctx context.Context, g Group, id string) ([]Entry, error) {
	cachedEntries, err := cachedOpHelper(List, g, id, func() (interface{}, error) {
		return g.List(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedEntries.([]Entry), nil
}

// CachedOpen caches a Readable object's Open method.
// When using the reader returned by this method, use idempotent read operations
// such as ReadAt or wrap it in a SectionReader. Using Read operations on the cached
// reader will change it and make subsequent uses of the cached reader invalid.
func CachedOpen(ctx context.Context, r Readable, id string) (SizedReader, error) {
	cachedContent, err := cachedOpHelper(Open, r, id, func() (interface{}, error) {
		return r.Open(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedContent.(SizedReader), nil
}

// CachedMetadata caches a Resource object's Metadata method
func CachedMetadata(ctx context.Context, r Resource, id string) (MetadataMap, error) {
	cachedMetadata, err := cachedOpHelper(Metadata, r, id, func() (interface{}, error) {
		return r.Metadata(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedMetadata.(MetadataMap), nil
}
