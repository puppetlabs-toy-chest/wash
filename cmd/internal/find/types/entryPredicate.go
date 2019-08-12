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
	SchemaRequired() bool
	RequireSchema()
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
		var schemaP EntrySchemaPredicate
		if entryP != nil {
			schemaP = entryP.SchemaP()
		}
		return schemaP, tokens, err
	}
}

type entryPredicate struct {
	p func(Entry) bool
	// Maintain a SchemaP object for the walker so that it only
	// traverses satisfying paths.
	schemaP EntrySchemaPredicate
	// schemaRequired should be true iff p will always return false
	// for schema-less entries. Informally, this is saying that
	// if a primary only make sense for entries with schemas (e.g. like
	// "kind"), then schemaRequired should be set to true. This way, if
	// the user runs something like `wash find -kind 'docker/*container'`,
	// the Walker will know not to traverse any schema-less plugins (since
	// the kinds of entries in those plugins is unknown).
	//
	// NOTE: The formal definition's necessary to prove the correctness
	// of schemaRequired in EntryPredicateAnd and EntryPredicateOr.
	schemaRequired bool
}

func (p1 *entryPredicate) P(e Entry) bool {
	if p1.SchemaRequired() && e.Schema == nil {
		// We will likely never hit this code-path since the Walker
		// won't visit schema-less entries when p1.SchemaRequired() == true.
		// However, it is still good to have this check here for correctness.
		return false
	}
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
		// schemaRequired shouldn't be negated because that is intrinsic
		// to the primary. For example, something like "! -kind '*dock*container'"
		// is parsed as "return anything that isn't a Docker container" so
		// it is still filtering on specific kinds of entries.
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

func (p1 *entryPredicate) SchemaRequired() bool {
	return p1.schemaRequired
}

func (p1 *entryPredicate) RequireSchema() {
	p1.schemaRequired = true
}

// EntryPredicateAnd represents an And operation on Entry predicates
type EntryPredicateAnd struct {
	*entryPredicate
	p1 EntryPredicate
	p2 EntryPredicate
}

// Combine implements predicate.BinaryOp#Combine
func (op *EntryPredicateAnd) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	ep1 := p1.(EntryPredicate)
	ep2 := p2.(EntryPredicate)
	return &EntryPredicateAnd{
		entryPredicate: &entryPredicate{
			schemaP: newEntrySchemaPredicateAnd(ep1.SchemaP(), ep2.SchemaP()),
			// Since p returns false iff ep1 OR ep2 return false, we can say that p will
			// always return false for schema-less entries iff ep1 OR ep2 require a schema.
			// Thus, p.schemaRequired == ep1.SchemaRequired() OR ep2.SchemaRequired().
			schemaRequired: ep1.SchemaRequired() || ep2.SchemaRequired(),
		},
		p1: ep1,
		p2: ep2,
	}
}

// P returns true if e satisfies the predicate, false otherwise
func (op *EntryPredicateAnd) P(e Entry) bool {
	return op.p1.P(e) && op.p2.P(e)
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (op *EntryPredicateAnd) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) && op.p2.IsSatisfiedBy(v)
}

// Negate returns Not(op)
func (op *EntryPredicateAnd) Negate() predicate.Predicate {
	return (&EntryPredicateOr{}).Combine(op.p1.Negate(), op.p2.Negate())
}

// EntryPredicateOr represents an Or operation on Entry predicates
type EntryPredicateOr struct {
	*entryPredicate
	p1 EntryPredicate
	p2 EntryPredicate
}

// Combine implements predicate.BinaryOp#Combine
func (op *EntryPredicateOr) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	ep1 := p1.(EntryPredicate)
	ep2 := p2.(EntryPredicate)
	return &EntryPredicateOr{
		entryPredicate: &entryPredicate{
			schemaP: newEntrySchemaPredicateOr(ep1.SchemaP(), ep2.SchemaP()),
			// Since p returns false iff ep1 OAND ep2 return false, we can say that p will
			// always return false for schema-less entries iff ep1 AND ep2 require a schema.
			// Thus, p.schemaRequired == ep1.SchemaRequired() AND ep2.SchemaRequired().
			schemaRequired: ep1.SchemaRequired() && ep2.SchemaRequired(),
		},
		p1: ep1,
		p2: ep2,
	}
}

// P returns true if e satisfies the predicate, false otherwise
func (op *EntryPredicateOr) P(e Entry) bool {
	return op.p1.P(e) || op.p2.P(e)
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (op *EntryPredicateOr) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) || op.p2.IsSatisfiedBy(v)
}

// Negate returns Not(op)
func (op *EntryPredicateOr) Negate() predicate.Predicate {
	return (&EntryPredicateAnd{}).Combine(op.p1.Negate(), op.p2.Negate())
}
