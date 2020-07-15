package rql

import (
	"context"
	"fmt"
	"sort"

	"github.com/puppetlabs/wash/plugin"
)

type walker interface {
	// Returns true if the walk is successful (i.e. does not
	// have any errors), false otherwise.
	Walk(ctx context.Context, start plugin.Entry) ([]Entry, error)
}

type walkerImpl struct {
	q    Query
	opts Options
}

// Make this a variable so that other tests can mock it
var newWalker = func(p Query, opts Options) walker {
	return &walkerImpl{
		q:    p,
		opts: opts,
	}
}

func (w *walkerImpl) Walk(ctx context.Context, start plugin.Entry) ([]Entry, error) {
	startEntry := newEntry(nil, start)
	startEntry.Path = ""
	s, err := plugin.Schema(start)
	if err != nil {
		return nil, err
	}
	if s != nil {
		schema := prune(newEntrySchema(s), w.q, w.opts)
		startEntry.Schema = schema
	}
	// TODO: Re-introduce something like SchemaRequired() so we can optimize
	// the traversal if w.q is a schema-predicate. See
	// https://github.com/puppetlabs/wash/blob/main/cmd/internal/find/walker.go#L47-L52
	entries, err := w.walk(ctx, &startEntry, 0)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// walk and visit take pointers because they update e's fields (like its Schema and
// Metadata)

func (w *walkerImpl) walk(ctx context.Context, e *Entry, depth int) ([]Entry, error) {
	entries := []Entry{}

	isStartEntry := e.Path == ""
	if !isStartEntry {
		// Visit the entry
		includeEntry, err := w.visit(ctx, e, depth)
		if err != nil {
			return nil, err
		} else if includeEntry {
			entries = append(entries, *e)
		}
	}

	childDepth := depth + 1
	if int(childDepth) <= w.opts.Maxdepth && e.Supports(plugin.ListAction()) {
		childrenMap, err := plugin.List(ctx, e.pluginEntry.(plugin.Parent))
		if err != nil {
			return nil, fmt.Errorf("could not get children of %v: %w\n", e.Path, err)
		} else {
			children := []Entry{}
			childrenMap.Range(func(cname string, childPluginEntry plugin.Entry) bool {
				child := newEntry(e, childPluginEntry)
				if e.SchemaKnown() {
					childSchema := e.Schema.GetChild(child.TypeID)
					if childSchema == nil {
						// Prune removed this child from the stree so that means
						// we do not need to walk it
						return true
					}
					child.Schema = childSchema
				}
				children = append(children, child)
				return true
			})
			// Sort the children by cname to ensure consistent ordering
			sort.Slice(children, func(i, j int) bool {
				return children[i].CName < children[j].CName
			})
			// Now walk the children
			for _, child := range children {
				descendants, err := w.walk(ctx, &child, childDepth)
				if err != nil {
					return nil, err
				}
				entries = append(entries, descendants...)
			}
		}
	}

	return entries, nil
}

func (w *walkerImpl) visit(ctx context.Context, e *Entry, depth int) (bool, error) {
	if depth < w.opts.Mindepth {
		return false, nil
	}
	if e.SchemaKnown() && !w.q.EvalEntrySchema(e.Schema) {
		// This is possible if e's a sibling/ancestor to a satisfying
		// node
		return false, nil
	}
	if w.opts.Fullmeta {
		// Fetch the entry's full metadata
		meta, err := plugin.Metadata(ctx, e.pluginEntry)
		if err != nil {
			return false, fmt.Errorf("could not get full metadata of %v: %w\n", e.Path, err)
		}
		e.Metadata = meta
	}
	return w.q.EvalEntry((*e)), nil
}
