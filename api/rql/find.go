package rql

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
)

// Find returns all descendants of the start entry that satisfy the given query
// Note that all returned entries' paths will start from "", which represents the
// start path. For example, if "childOne" and "childTwo" are the cnames of the start
// entry's children, then their "Path" fields will be set to "childOne" and "childTwo"
// (where the start entry's path of "" is automatically prefixed).
//
// Each entry's children are descended in lexicographic order (based on their cnames).
// So given entries "foo", "foo/bar", "foo/baz", "foo/baz/1", the returned entries will
// be ["foo", "foo/bar", "foo/baz", "foo/baz/1"] (because "bar" comes before "baz").
func Find(ctx context.Context, start plugin.Entry, query Query, options Options) ([]Entry, error) {
	return newWalker(query, options).Walk(ctx, start)
}
