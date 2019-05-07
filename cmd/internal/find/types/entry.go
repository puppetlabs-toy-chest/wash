package types

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	apitypes "github.com/puppetlabs/wash/api/types"
)

// Entry represents an Entry as interpreted by `wash find`
type Entry struct {
	apitypes.Entry
	NormalizedPath string
}

// EntryPredicate represents a predicate on a Wash entry.
type EntryPredicate func(Entry) bool

// And returns p1 && p2
func (p1 EntryPredicate) And(p2 predicate.Predicate) predicate.Predicate {
	return EntryPredicate(func(e Entry) bool {
		return p1(e) && (p2.(EntryPredicate))(e)
	})
}

// Or returns p1 || p2
func (p1 EntryPredicate) Or(p2 predicate.Predicate) predicate.Predicate {
	return EntryPredicate(func(e Entry) bool {
		return p1(e) || (p2.(EntryPredicate))(e)
	})
}

// Negate returns Not(p1)
func (p1 EntryPredicate) Negate() predicate.Predicate {
	return EntryPredicate(func(e Entry) bool {
		return !p1(e)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 EntryPredicate) IsSatisfiedBy(v interface{}) bool {
	entry, ok := v.(Entry)
	if !ok {
		return false
	}
	return p1(entry)
}

// EntryPredicateParser parses Entry predicates
type EntryPredicateParser func(tokens []string) (EntryPredicate, []string, error)

// Parse parses an EntryPredicate from the given input.
func (parser EntryPredicateParser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return parser(tokens)
}
