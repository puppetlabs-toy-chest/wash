package types

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// EntrySchemaPredicate represents a predicate on an entry's schema
type EntrySchemaPredicate interface {
	predicate.Predicate
	P(*EntrySchema) bool
}

// ToEntrySchemaP converts p to an EntrySchemaPredicate object
func ToEntrySchemaP(p func(*EntrySchema) bool) EntrySchemaPredicate {
	return entrySchemaPredicate(p)
}

// entrySchemaPredicate represents a predicate on an entry's schema
type entrySchemaPredicate func(*EntrySchema) bool

func (p1 entrySchemaPredicate) P(s *EntrySchema) bool {
	return p1(s)
}

// Negate returns Not(p1)
func (p1 entrySchemaPredicate) Negate() predicate.Predicate {
	return entrySchemaPredicate(func(s *EntrySchema) bool {
		return !p1(s)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 entrySchemaPredicate) IsSatisfiedBy(v interface{}) bool {
	s, ok := v.(*EntrySchema)
	if !ok {
		return false
	}
	return p1(s)
}

// EntrySchemaPredicateParser parses EntrySchema predicates
type EntrySchemaPredicateParser func(tokens []string) (EntrySchemaPredicate, []string, error)

// Parse parses an entrySchemaPredicate from the given input.
func (parser EntrySchemaPredicateParser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return parser(tokens)
}
