package types

import (
	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// EntrySchema is a wrapper to apitypes.EntrySchema
type EntrySchema apitypes.EntrySchema

// EntrySchemaPredicate represents a predicate on an entry's schema
type EntrySchemaPredicate func(EntrySchema) bool

// And returns p1 && p2
func (p1 EntrySchemaPredicate) And(p2 predicate.Predicate) predicate.Predicate {
	return EntrySchemaPredicate(func(s EntrySchema) bool {
		return p1(s) && (p2.(EntrySchemaPredicate))(s)
	})
}

// Or returns p1 || p2
func (p1 EntrySchemaPredicate) Or(p2 predicate.Predicate) predicate.Predicate {
	return EntrySchemaPredicate(func(s EntrySchema) bool {
		return p1(s) || (p2.(EntrySchemaPredicate))(s)
	})
}

// Negate returns Not(p1)
func (p1 EntrySchemaPredicate) Negate() predicate.Predicate {
	return EntrySchemaPredicate(func(s EntrySchema) bool {
		return !p1(s)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 EntrySchemaPredicate) IsSatisfiedBy(v interface{}) bool {
	s, ok := v.(EntrySchema)
	if !ok {
		return false
	}
	return p1(s)
}
