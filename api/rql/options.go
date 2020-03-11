package rql

// Options represent the RQL's options
type Options struct {
	// Mindepth is the minimum depth. Descendants at lesser depths are not included
	// in the RQL's returned list of entries.
	//
	// Depth starts from 0. For example, given paths "foo", "foo/bar", "foo/bar/baz",
	// assume "foo" is the start path. Then "foo" is at depth 0, "foo/bar" is at depth 1,
	// "foo/bar/baz" is at depth 2, etc.
	Mindepth int
	// Maxdepth is the maximum depth. Descendants at greater depths are not included
	// in the RQL's returned list of entries. See Mindepth's comments to understand how
	// depth is calculated.
	Maxdepth int
	// Fullmeta is short for "full metadata". If set, then meta primary queries act on
	// the entry's full metadata, and the returned list of entries will include the entry's
	// full metadata. If unset then the RQL uses the partial metadata instead.
	//
	// Note that setting Fullmeta could result in O(N) extra requests to fetch the metadata,
	// where N is the number of visited entries. Using the partial metadata (unsetting Fullmeta)
	// does not result in any extra request.
	Fullmeta bool
}

// DefaultMaxdepth is the default value of the maxdepth option.
// It is set to the max value of a 32-bit integer.
const DefaultMaxdepth = 1<<31 - 1

// NewOptions creates a new Options object
func NewOptions() Options {
	return Options{
		Mindepth: 0,
		Maxdepth: DefaultMaxdepth,
		Fullmeta: false,
	}
}
