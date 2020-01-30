package rql

import (
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

// Primary is the interface implemented by all the primaries. A primary's
// domain is the set of all entries that it applies to. The domain can be
// specified at the instance-level (EntryInDomain), the schema-level
// (EntrySchemaInDomain) or both.
//
// Note that a primary can either be an EntryPredicate, an EntrySchemaPredicate,
// both or neither. In practice, the RQL will use EvalEntrySchema when pruning the
// stree and EvalEntry when filtering the entries. EntryInDomain and EntrySchemaInDomain
// are only needed to correctly negate the primaries (for without it, strict negation
// would return true for entries that are outside the primary's domain).
//
// For a given primary p, here are the possible scenarios for EvalEntrySchema
// and EvalEntry (including negation). Note that p.EntrySchemaInDomain and
// p.EvalEntrySchema can assume that s != nil, and that evaluation order is from
// left-to-right (so EntrySchemaInDomain first then EvalEntrySchema) with
// appropriate '&&' short-circuiting.
//   * If p implements EntrySchemaPredicate, then for RQL
//         EvalEntrySchema(s) == p.EntrySchemaInDomain(s) && p.EvalEntrySchema(s)
//         NOT(EvalEntrySchema(s)) == p.EntrySchemaInDomain(s) && !p.EvalEntrySchema(s)
//     otherwise
//         EvalEntrySchema(s) == p.EntrySchemaInDomain(s)
//         NOT(EvalEntrySchema(s)) == p.EntrySchemaInDomain(s)
//
//   * If p implements EntryPredicate, then for RQL
//         EvalEntry(e) == p.EntryInDomain(e) && p.EvalEntry(e)
//         NOT(EvalEntry(e)) == p.EntryInDomain(e) && !p.EvalEntry(e)
//     otherwise
//         EvalEntry(e) == p.EntryInDomain(e)
//         NOT(EvalEntry(e)) == p.EntryInDomain(e)
//
type Primary interface {
	ASTNode
	EntryInDomain(Entry) bool
	EntrySchemaInDomain(*EntrySchema) bool
}

// EntryPredicate represents a predicate on an entry
type EntryPredicate interface {
	Primary
	EvalEntry(Entry) bool
}

// EntrySchemaPredicate represents a predicate on an entry schema object
type EntrySchemaPredicate interface {
	Primary
	EvalEntrySchema(*EntrySchema) bool
}

// ValuePredicate represents a predicate on a metadata (JSON) value. Its
// domain is the set of all value types that the predicate applies to.
//
// In practice, the RQL will use EvalValue; ValueInDomain's only needed
// to correctly negate the predicates (like EntryInDomain/EntrySchemaInDomain).
// Here are the semantics for a given predicate. Note that evaluation order
// is from left-to-right (so ValueInDomain first then EvalValue) with appropriate
// '&&' short-circuiting.
//     EvalValue(v) == p.ValueInDomain(v) && p.EvalValue(v)
//     NOT(EvalValue(v)) == p.ValueInDomain(v) && !p.EvalValue(v)
type ValuePredicate interface {
	ASTNode
	ValueInDomain(interface{}) bool
	EvalValue(interface{}) bool
}

// StringPredicate represents a predicate on a string value
type StringPredicate interface {
	ASTNode
	EvalString(string) bool
}

// NumericPredicate represents a predicate on a numeric value. The
// decimal.Decimal type lets us handle arbitrarily large numbers.
type NumericPredicate interface {
	ASTNode
	EvalNumeric(decimal.Decimal) bool
}

// TimePredicate represents a predicate on a time value.
type TimePredicate interface {
	ASTNode
	EvalTime(time.Time) bool
}

// ActionPredicate represents a predicate on a Wash action.
type ActionPredicate interface {
	ASTNode
	EvalAction(plugin.Action) bool
}
