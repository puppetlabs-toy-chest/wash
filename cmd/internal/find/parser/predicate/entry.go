package predicate

import "github.com/puppetlabs/wash/cmd/internal/find/types"

// Entry represents a predicate on a Wash entry.
type Entry func(types.Entry) bool

// And returns p1 && p2
func (p1 Entry) And(p2 Predicate) Predicate {
	return Entry(func(e types.Entry) bool {
		return p1(e) && (p2.(Entry))(e)
	})
}

// Or returns p1 || p2
func (p1 Entry) Or(p2 Predicate) Predicate {
	return Entry(func(e types.Entry) bool {
		return p1(e) || (p2.(Entry))(e)
	})
}

// Negate returns Not(p1)
func (p1 Entry) Negate() Predicate {
	return Entry(func(e types.Entry) bool {
		return !p1(e)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 Entry) IsSatisfiedBy(v interface{}) bool {
	entry, ok := v.(types.Entry)
	if !ok {
		return false
	}
	return p1(entry)
}

// EntryParser is a type that parses Entry predicates
type EntryParser func(tokens []string) (Entry, []string, error)

// Parse parses an Entry predicate from the given input.
func (parser EntryParser) Parse(tokens []string) (Predicate, []string, error) {
	return parser(tokens)
}