package types

import (
	"flag"
	"io/ioutil"

	"github.com/puppetlabs/wash/cmd/util"
)

// Options represents the find command's options.
type Options struct {
	Depth    bool
	Mindepth uint
	Maxdepth int
	Help HelpOption
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
	fs.SetOutput(ioutil.Discard)
	fs.BoolVar(&opts.Depth, DepthFlag, opts.Depth, "")
	fs.UintVar(&opts.Mindepth, MindepthFlag, opts.Mindepth, "")
	fs.IntVar(&opts.Maxdepth, MaxdepthFlag, opts.Maxdepth, "")
	return fs
}

// OptionsTable returns a table containing all of `wash find`'s available
// options
func OptionsTable() *cmdutil.Table {
	return cmdutil.NewTable(
		[]string{"Flags:",                 ""},
		[]string{"      -depth",           "Visit the children first before the parent (default false)"},
		[]string{"      -mindepth levels", "Do not print entries at levels less than levels (default 0)"},
		[]string{"      -maxdepth levels", "Do not print entries at levels greater than levels (default infinity)"},
		[]string{"  -h, -help",            "Print the usage"},
		[]string{"  -h, -help <primary>",  "Print a detailed description of the specified primary"},
		[]string{"  -h, -help syntax",     "Print a detailed description of find's expression syntax"},
	)
}

// HelpOption represents the -help option. If HasValue is set, then
// that means the input was "-help <primary>|syntax". In that case,
// only one of Primary/Syntax is set. Otherwise, the input was "-help".
//
// See the comments in parser.parseOptions for more details on why
// this does not implement the Value interface.
type HelpOption struct {
	Requested bool
	HasValue bool
	// Cannot use *primary.Primary here b/c doing so would introduce
	// an import cycle. Resolving that import cycle for a slightly
	// cleaner implementation is not worth the additional complexity
	// associated with introducing more fine-grained packages.
	Primary string
	Syntax bool
}

