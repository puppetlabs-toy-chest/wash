package types

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// EntryPredicate represents a predicate on a Wash entry.
type EntryPredicate struct {
	P func(Entry) bool
	// Maintain a SchemaP object for the walker so that it only
	// traverses satisfying paths.
	SchemaP EntrySchemaPredicate
}

// ToEntryP converts p to an EntryPredicate object
func ToEntryP(p func(e Entry) bool) *EntryPredicate {
	return &EntryPredicate{
		P: p,
		SchemaP: func(s *EntrySchema) bool {
			return true
		},
	}
}

// And returns p1 && p2
func (p1 *EntryPredicate) And(p2 predicate.Predicate) predicate.Predicate {
	ep2 := p2.(*EntryPredicate)
	return &EntryPredicate{
		P: func(e Entry) bool {
			return p1.P(e) && ep2.P(e)
		},
		SchemaP: p1.SchemaP.And(ep2.SchemaP).(EntrySchemaPredicate),
	}
}

// Or returns p1 || p2
func (p1 *EntryPredicate) Or(p2 predicate.Predicate) predicate.Predicate {
	ep2 := p2.(*EntryPredicate)
	return &EntryPredicate{
		P: func(e Entry) bool {
			return p1.P(e) || ep2.P(e)
		},
		SchemaP: p1.SchemaP.Or(ep2.SchemaP).(EntrySchemaPredicate),
	}
}

// Negate returns Not(p1)
func (p1 EntryPredicate) Negate() predicate.Predicate {
	return &EntryPredicate{
		P: func(e Entry) bool {
			return !p1.P(e)
		},
		SchemaP: p1.SchemaP.Negate().(EntrySchemaPredicate),
	}
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 *EntryPredicate) IsSatisfiedBy(v interface{}) bool {
	entry, ok := v.(Entry)
	if !ok {
		return false
	}
	return p1.P(entry)
}

// EntryPredicateParser parses Entry predicates
type EntryPredicateParser func(tokens []string) (*EntryPredicate, []string, error)

// Parse parses an EntryPredicate from the given input.
func (parser EntryPredicateParser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return parser(tokens)
}

// ToSchemaPParser converts parser to a schema predicate parser. This is mostly
// used by the tests
func (parser EntryPredicateParser) ToSchemaPParser() EntrySchemaPredicateParser {
	return func(tokens []string) (EntrySchemaPredicate, []string, error) {
		entryP, tokens, err := parser(tokens)
		if err != nil {
			return nil, tokens, err
		}
		return entryP.SchemaP, tokens, nil
	}
}
