package types

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// EntrySchemaPredicate represents a predicate on an entry's schema.
// Only implement this interface if your schema predicate has specialized
// negation semantics. See the meta primary's schema predicate for
// an example.
type EntrySchemaPredicate interface {
	predicate.Predicate
	P(*EntrySchema) bool
}

// ToEntrySchemaP converts p to an EntrySchemaPredicate object
func ToEntrySchemaP(p func(*EntrySchema) bool) EntrySchemaPredicate {
	return entrySchemaPredicateFunc(p)
}

// entrySchemaPredicateFunc implements the EntrySchemaPredicate
// interface for the corresponding function type
type entrySchemaPredicateFunc func(*EntrySchema) bool

func (p1 entrySchemaPredicateFunc) P(s *EntrySchema) bool {
	return p1(s)
}

// Negate returns Not(p1)
func (p1 entrySchemaPredicateFunc) Negate() predicate.Predicate {
	return entrySchemaPredicateFunc(func(s *EntrySchema) bool {
		return !p1(s)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 entrySchemaPredicateFunc) IsSatisfiedBy(v interface{}) bool {
	s, ok := v.(*EntrySchema)
	if !ok {
		return false
	}
	return p1(s)
}

// EntrySchemaPredicateParser parses EntrySchema predicates
type EntrySchemaPredicateParser func(tokens []string) (EntrySchemaPredicate, []string, error)

// Parse parses an entrySchemaPredicateFunc from the given input.
func (parser EntrySchemaPredicateParser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return parser(tokens)
}

// Note that we need the EntrySchemaPAnd/EntrySchemaPOr classes to ensure that
// De'Morgan's law is enforced. Also, Combine for these classes is not
// implemented b/c it is not needed -- EntryPredicateAnd/EntryPredicateOr's combine
// handles schema predicates.

type entrySchemaPredicateAnd struct {
	p1 EntrySchemaPredicate
	p2 EntrySchemaPredicate
}

func newEntrySchemaPredicateAnd(p1 EntrySchemaPredicate, p2 EntrySchemaPredicate) *entrySchemaPredicateAnd {
	return &entrySchemaPredicateAnd{
		p1: p1,
		p2: p2,
	}
}

func (op *entrySchemaPredicateAnd) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) && op.p2.IsSatisfiedBy(v)
}

func (op *entrySchemaPredicateAnd) Negate() predicate.Predicate {
	return newEntrySchemaPredicateOr(op.p1.Negate().(EntrySchemaPredicate), op.p2.Negate().(EntrySchemaPredicate))
}

func (op *entrySchemaPredicateAnd) P(s *EntrySchema) bool {
	return op.p1.P(s) && op.p2.P(s)
}

type entrySchemaPredicateOr struct {
	p1 EntrySchemaPredicate
	p2 EntrySchemaPredicate
}

func newEntrySchemaPredicateOr(p1 EntrySchemaPredicate, p2 EntrySchemaPredicate) *entrySchemaPredicateOr {
	return &entrySchemaPredicateOr{
		p1: p1,
		p2: p2,
	}
}

func (op *entrySchemaPredicateOr) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) || op.p2.IsSatisfiedBy(v)
}

func (op *entrySchemaPredicateOr) Negate() predicate.Predicate {
	return newEntrySchemaPredicateAnd(op.p1.Negate().(EntrySchemaPredicate), op.p2.Negate().(EntrySchemaPredicate))
}

func (op *entrySchemaPredicateOr) P(s *EntrySchema) bool {
	return op.p1.P(s) || op.p2.P(s)
}
