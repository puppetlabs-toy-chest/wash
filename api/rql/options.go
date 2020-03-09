package rql

// Options represent the RQL's options
type Options struct {
	Mindepth int
	Maxdepth int
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
