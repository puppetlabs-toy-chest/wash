package types

import (
	"flag"
	"io/ioutil"
)

// Options represents the find command's options.
type Options struct {
	Depth    bool
	Mindepth uint
	Maxdepth int
	setFlags map[string]struct{}
}

// DefaultMaxdepth is the default value of the maxdepth option.
// It is set to the max value of a 32-bit integer.
const DefaultMaxdepth = 1<<31 - 1

// NewOptions creates a new Options object
func NewOptions() Options {
	return Options{
		Depth:    false,
		Mindepth: 0,
		// We make Maxdepth an int because of the `meta` primary.
		// See the comments in `primary/meta.go` for more details.
		Maxdepth: DefaultMaxdepth,
		setFlags: make(map[string]struct{}),
	}
}

const (
	// DepthFlag is the name of the depth option's flag
	DepthFlag = "depth"
	// MindepthFlag is the name of the mindepth option's flag
	MindepthFlag = "mindepth"
	// MaxdepthFlag is the name of the maxdepth option's flag
	MaxdepthFlag = "maxdepth"
)

// IsSet returns true if the flag was set, false otherwise.
func (opts *Options) IsSet(flag string) bool {
	_, ok := opts.setFlags[flag]
	return ok
}

// MarkAsSet marks the flag as set.
func (opts *Options) MarkAsSet(flag string) {
	opts.setFlags[flag] = struct{}{}
}

// FlagSet returns a flagset representing
// opts.
func (opts *Options) FlagSet() *flag.FlagSet {
	// Create the flag-set that's tied to our options.
	fs := flag.NewFlagSet("options", flag.ContinueOnError)
	// TODO: Handle usage later
	fs.Usage = func() {}
	fs.SetOutput(ioutil.Discard)
	fs.BoolVar(&opts.Depth, DepthFlag, opts.Depth, "")
	fs.UintVar(&opts.Mindepth, MindepthFlag, opts.Mindepth, "")
	fs.IntVar(&opts.Maxdepth, MaxdepthFlag, opts.Maxdepth, "")
	return fs
}
