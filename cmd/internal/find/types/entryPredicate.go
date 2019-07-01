package types

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// EntryPredicate represents a predicate on a Wash entry.
type EntryPredicate interface {
	predicate.Predicate
	P(Entry) bool
	SchemaP() EntrySchemaPredicate
	SetSchemaP(EntrySchemaPredicate)
}

// ToEntryP converts p to an EntryPredicate object
func ToEntryP(p func(e Entry) bool) EntryPredicate {
	return &entryPredicate{
		p: p,
		schemaP: ToEntrySchemaP(func(s *EntrySchema) bool {
			return true
		}),
	}
}

// EntryPredicateParser parses Entry predicates
type EntryPredicateParser func(tokens []string) (EntryPredicate, []string, error)

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
		return entryP.SchemaP(), tokens, nil
	}
}

type entryPredicate struct {
	p func(Entry) bool
	// Maintain a SchemaP object for the walker so that it only
	// traverses satisfying paths.
	schemaP EntrySchemaPredicate
}

func (p1 *entryPredicate) P(e Entry) bool {
	return p1.p(e)
}

func (p1 *entryPredicate) SchemaP() EntrySchemaPredicate {
	return p1.schemaP
}

func (p1 *entryPredicate) SetSchemaP(schemaP EntrySchemaPredicate) {
	p1.schemaP = schemaP
}

// Negate returns Not(p1)
func (p1 *entryPredicate) Negate() predicate.Predicate {
	return &entryPredicate{
		p: func(e Entry) bool {
			return !p1.P(e)
		},
		schemaP: p1.SchemaP().Negate().(EntrySchemaPredicate),
	}
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 *entryPredicate) IsSatisfiedBy(v interface{}) bool {
	entry, ok := v.(Entry)
	if !ok {
		return false
	}
	return p1.P(entry)
}

// EntryPredicateAnd represents an And operation on Entry predicates
type EntryPredicateAnd struct {
	*entryPredicate
}

// Combine implements predicate.BinaryOp#Combine
func (op *EntryPredicateAnd) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	ep1 := p1.(EntryPredicate)
	ep2 := p2.(EntryPredicate)
	return &EntryPredicateAnd{
		entryPredicate: &entryPredicate{
			p: func(e Entry) bool {
				return ep1.P(e) && ep2.P(e)
			},
			schemaP: ToEntrySchemaP(func(s *EntrySchema) bool {
				return ep1.SchemaP().P(s) && ep2.SchemaP().P(s)
			}),
		},
	}
}

// EntryPredicateOr represents an Or operation on Entry predicates
type EntryPredicateOr struct {
	*entryPredicate
}

// Combine implements predicate.BinaryOp#Combine
func (op *EntryPredicateOr) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	ep1 := p1.(EntryPredicate)
	ep2 := p2.(EntryPredicate)
	return &EntryPredicateOr{
		entryPredicate: &entryPredicate{
			p: func(e Entry) bool {
				return ep1.P(e) || ep2.P(e)
			},
			schemaP: ToEntrySchemaP(func(s *EntrySchema) bool {
				return ep1.SchemaP().P(s) || ep2.SchemaP().P(s)
			}),
		},
	}
}
