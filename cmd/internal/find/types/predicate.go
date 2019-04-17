package types

// Predicate represents a predicate used by `wash find`
type Predicate func(e Entry) bool
