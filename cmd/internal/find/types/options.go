package types

import (
	"flag"
	"io/ioutil"
)

// Options represents the find command's options.
type Options struct {
	Depth    bool
	Mindepth uint
	Maxdepth uint
	setFlags map[string]struct{}
}

// NewOptions creates a new Options object
func NewOptions() Options {
	return Options{
		Depth:    false,
		Mindepth: 0,
		Maxdepth: ^uint(0),
		setFlags: make(map[string]struct{}),
	}
}

// DepthFlag is the name of the depth option's flag
var DepthFlag = "depth"

// MindepthFlag is the name of the mindepth option's flag
var MindepthFlag = "mindepth"

// MaxdepthFlag is the name of the maxdepth option's flag
var MaxdepthFlag = "maxdepth"

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
	fs.UintVar(&opts.Maxdepth, MaxdepthFlag, opts.Maxdepth, "")
	return fs
}
