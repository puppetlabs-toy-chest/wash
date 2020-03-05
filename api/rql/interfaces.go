package rql

import (
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

// Primary represents the interface implemented by all the primaries
type Primary interface {
	ASTNode
	IsPrimary() bool
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

// ValuePredicate represents a predicate on a metadata (JSON) value
type ValuePredicate interface {
	ASTNode
	EvalValue(interface{}) bool
	EvalValueSchema(*plugin.JSONSchema) bool
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
