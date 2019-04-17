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
}

// NewOptions creates a new Options object
func NewOptions() Options {
	return Options{
		Depth:    false,
		Mindepth: 0,
		Maxdepth: ^uint(0),
	}
}

// FlagSet returns a flagset representing
// opts.
func (opts *Options) FlagSet() *flag.FlagSet {
	// Create the flag-set that's tied to our options.
	fs := flag.NewFlagSet("options", flag.ContinueOnError)
	// TODO: Handle usage later
	fs.Usage = func() {}
	fs.SetOutput(ioutil.Discard)
	fs.BoolVar(&opts.Depth, "depth", opts.Depth, "")
	fs.UintVar(&opts.Mindepth, "mindepth", opts.Mindepth, "")
	fs.UintVar(&opts.Maxdepth, "maxdepth", opts.Maxdepth, "")
	return fs
}
