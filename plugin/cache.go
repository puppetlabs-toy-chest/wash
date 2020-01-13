package plugin

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/puppetlabs/wash/datastore"
)

// KeyType is used to create a unique key type for looking up context values.
type keyType int

// id is used to identify the parent's ID in a context.
const parentID keyType = iota

var cache datastore.Cache

// InitCache initializes the cache
func InitCache() {
	if notRunningTests() {
		cache = datastore.NewMemCache()
	} else {
		panic("InitCache can only be called in production. Tests should call SetTestCache instead.")
	}
}

// SetTestCache sets the cache to the provided mock. It can only be called by the tests.
// Returns a context that includes a parent ID so later cache operations will succeed.
func SetTestCache(c datastore.Cache) context.Context {
	if notRunningTests() {
		panic("SetTestCache can only be called when running the tests")
	}

	if cache != nil {
		panic("The test cache has already been set")
	}

	cache = c
	return context.WithValue(context.Background(), parentID, "/")
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

var opNameRegex = regexp.MustCompile("^[a-zA-Z]+$")

const opQualifier = "^[a-zA-Z]+::"

// opKeyRegex returns a regex that matches <op>::<path>
func opKeyRegex(op string, path string) *regexp.Regexp {
	opRegex := "^" + regexp.QuoteMeta(op+"::")

	var expr string
	if path == "/" {
		expr = opRegex + "/$"
	} else {
		expr = opRegex + "/" + regexp.QuoteMeta(strings.Trim(path, "/")) + "$"
	}

	return regexp.MustCompile(expr)
}

// This returns a regex that matches <op>::<path> and <op>::<child_path>
// where <child_path> is a descendant of <path>.
func allOpKeysIncludingChildrenRegex(path string) *regexp.Regexp {
	var expr string
	if path == "/" {
		expr = opQualifier + "/.*"
	} else {
		expr = opQualifier + "/" + regexp.QuoteMeta(strings.Trim(path, "/")) + "($|/.*)"
	}

	return regexp.MustCompile(expr)
}

// ClearCacheFor removes entries from the cache that match or are children of the provided path.
// If successful, returns an array of deleted keys. Optionally clear the list operation for the
// parent to remove any attributes related to the specified entry.
//
// TODO: If path == "/", we could optimize this by calling cache.Flush(). Not important
// right now, but may be worth considering in the future.
func ClearCacheFor(path string, clearParentList bool) []string {
	rx := allOpKeysIncludingChildrenRegex(path)
	deleted := cache.Delete(rx)

	if clearParentList {
		parentID, _ := splitID(path)
		listOpName := defaultOpCodeToNameMap[ListOp]
		deleted = append(deleted, cache.Delete(opKeyRegex(listOpName, parentID))...)
	}

	return deleted
}

// returns (parentID, cname)
func splitID(entryID string) (string, string) {
	segments := strings.Split(entryID, "/")
	parentID := strings.Join(segments[:len(segments)-1], "/")
	cname := segments[len(segments)-1]
	return parentID, cname
}

type opFunc func() (interface{}, error)

// CachedOp caches the given op's result for the duration specified by the
// ttl. You should use it when you need more fine-grained caching than what
// the existing CachedList, CachedOpen, and CachedMetadata methods provide.
// For example, CachedOp could be useful to cache an API request whose
// response lets you implement Open() and Metadata() for the given entry.
//
// A ttl of 0 uses the cache default of 1 minute. Negative ttls are not allowed.
//
// CachedOp uses the supplied context to determine where to log activity.
func CachedOp(ctx context.Context, opName string, entry Entry, ttl time.Duration, op opFunc) (interface{}, error) {
	if !opNameRegex.MatchString(opName) {
		panic(fmt.Sprintf("The opName %v does not match %v", opName, opNameRegex.String()))
	}

	for _, actionOpName := range defaultOpCodeToNameMap {
		if opName == actionOpName {
			panic(fmt.Sprintf("The opName %v conflicts with Cached%v", opName, actionOpName))
		}
	}

	if ttl < 0 {
		panic("plugin.CachedOp: received a negative TTL")
	}

	return cachedOp(ctx, opName, entry, ttl, op)
}

// DuplicateCNameErr represents a duplicate cname error, which
// occurs when at least two children have the same cname.
type DuplicateCNameErr struct {
	ParentID                 string
	FirstChildName           string
	FirstChildSlashReplacer  rune
	SecondChildName          string
	SecondChildSlashReplacer rune
	CName                    string
}

func (c DuplicateCNameErr) Error() string {
	pluginName := strings.SplitN(
		strings.TrimLeft(c.ParentID, "/"),
		"/",
		2,
	)[0]

	return fmt.Sprintf(
		"error listing %v: children %v and %v have the same cname of %v. This means that either the %v plugin's API returns duplicate names, or that you need to use a different slash replacer (see EntryBase#SetSlashReplacer)",
		c.ParentID,
		c.FirstChildName,
		c.SecondChildName,
		c.CName,
		pluginName,
	)
}

// CachedList caches a Parent's List method. It also sets the
// children's IDs to <parent_id> + "/" + <child_cname>.
//
// CachedList returns a map of <entry_cname> => <entry_object> to optimize
// querying a specific entry.
func cachedList(ctx context.Context, p Parent) (*EntryMap, error) {
	cachedEntries, err := cachedDefaultOp(ctx, ListOp, p, func() (interface{}, error) {
		// Including the entry's ID allows plugin authors to use any Cached* methods defined on the
		// children after their creation. This is necessary when the child's Cached* methods are used
		// to calculate its attributes. Note that the child's ID is set in cachedOp.
		entries, err := p.List(context.WithValue(ctx, parentID, p.eb().id))
		if err != nil {
			return nil, err
		}

		searchedEntries := newEntryMap()
		for _, entry := range entries {
			cname := CName(entry)

			if duplicateEntry, ok := searchedEntries.mp[cname]; ok {
				return nil, DuplicateCNameErr{
					ParentID:                 p.eb().id,
					FirstChildName:           duplicateEntry.eb().name,
					FirstChildSlashReplacer:  duplicateEntry.eb().slashReplacer,
					SecondChildName:          entry.eb().name,
					SecondChildSlashReplacer: entry.eb().slashReplacer,
					CName:                    cname,
				}
			}

			if entry.eb().isInaccessible {
				// Skip entries that are expected to be inaccessible.
				continue
			}

			searchedEntries.mp[cname] = entry

			// Ensure ID is set on all entries so that we can use it for caching later in places
			// where the context doesn't include the parent's ID.
			setChildID(p.eb().id, entry)

			passAlongWrappedTypes(p, entry)
		}

		return searchedEntries, nil
	})

	if err != nil {
		return nil, err
	}

	return cachedEntries.(*EntryMap), nil
}

// cachedRead caches an entry's Read method
func cachedRead(ctx context.Context, e Entry) (entryContent, error) {
	cachedContent, err := cachedDefaultOp(ctx, ReadOp, e, func() (interface{}, error) {
		switch signature := ReadAction().signature(e); signature {
		case defaultSignature:
			// Both external and core plugin entries that have the default Read signature
			// implement the Readable interface, so we can go ahead and cast directly.
			r := e.(Readable)
			rawContent, err := r.Read(ctx)
			if err != nil {
				return nil, err
			}
			return newEntryContent(rawContent), nil
		case blockReadableSignature:
			var readFunc blockReadFunc
			switch t := e.(type) {
			case externalPlugin:
				readFunc = func(ctx context.Context, size int64, offset int64) ([]byte, error) {
					return t.blockRead(ctx, size, offset)
				}
			case BlockReadable:
				readFunc = func(ctx context.Context, size int64, offset int64) ([]byte, error) {
					return t.Read(ctx, size, offset)
				}
			default:
				// We should never hit this code-path
				panic("attempting to retrieve the content of a non-readable entry")
			}
			content := newBlockReadableEntryContent(readFunc)
			if attr := e.eb().attributes; attr.HasSize() {
				content.sz = attr.Size()
			}
			return content, nil
		default:
			// We should never hit this code-path
			msg := fmt.Sprintf("unknown signature '%v' for read", signature)
			panic(msg)
		}
	})

	if err != nil {
		return nil, err
	}

	return cachedContent.(entryContent), nil
}

// cachedMetadata caches an entry's Metadata method
func cachedMetadata(ctx context.Context, e Entry) (JSONObject, error) {
	cachedMetadata, err := cachedDefaultOp(ctx, MetadataOp, e, func() (interface{}, error) {
		return e.Metadata(ctx)
	})

	if err != nil {
		return nil, err
	}

	return cachedMetadata.(JSONObject), nil
}

// Common helper for CachedList, CachedOpen and CachedMetadata
func cachedDefaultOp(ctx context.Context, opCode defaultOpCode, entry Entry, op opFunc) (interface{}, error) {
	opName := defaultOpCodeToNameMap[opCode]
	ttl := entry.eb().ttl[opCode]

	return cachedOp(ctx, opName, entry, ttl, op)
}

// Common helper for CachedOp and cachedDefaultOp.
func cachedOp(ctx context.Context, opName string, entry Entry, ttl time.Duration, op opFunc) (interface{}, error) {
	if cache == nil {
		if notRunningTests() {
			panic("The cache was not initialized. You can initialize the cache by invoking plugin.InitCache()")
		} else {
			panic("The test cache was not set. You can set it by invoking plugin.SetTestCache(<cache>)")
		}
	}

	if ttl < 0 {
		return op()
	}

	if entry.eb().id == "" {
		// Try to set the ID based on parent ID
		if obj := ctx.Value(parentID); obj != nil {
			setChildID(obj.(string), entry)
		} else {
			panic(fmt.Sprintf("Cached op %v on %v had no cache ID and context did not include parent ID", opName, entry.eb().name))
		}
	}

	return cache.GetOrUpdate(opName, entry.eb().id, ttl, false, op)
}

func setChildID(parentID string, child Entry) {
	id := strings.TrimRight(parentID, "/") + "/" + CName(child)
	child.eb().id = id
}
