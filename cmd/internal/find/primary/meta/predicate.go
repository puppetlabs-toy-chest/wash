package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

/*
Predicate => ObjectPredicate |
             ArrayPredicate  |
             PrimitivePredicate
*/
func parsePredicate(tokens []string) (predicate.Predicate, []string, error) {
	cp := &predicate.CompositeParser{
		MatchErrMsg: "expected either a primitive, object, or array predicate",
		Parsers: []predicate.Parser{
			predicate.ToParser(parseObjectPredicate),
			predicate.ToParser(parseArrayPredicate),
			predicate.ToParser(parsePrimitivePredicate),
		},
	}
	return cp.Parse(tokens)
}

// genericPredicate represents a `meta` primary predicate "base" class.
// Child classes should only override the Negate method. Here's why:
//   * Some `meta` primary predicates perform type validation, returning
//     false for a mis-typed value. genericPredicate#Negate is a strict
//     negation, so it will return true for a mis-typed value. This is bad.
//
//   * Some of the more complicated predicates require additional negation
//     semantics. For example, ObjectPredicate returns false if the key does
//     not exist. A negated ObjectPredicate should also return false for this
//     case.
//
// Both of these issues are resolved if the child class overrides Negate.
type genericPredicate func(interface{}) bool

// And returns p1 && p2
func (p1 genericPredicate) And(p2 predicate.Predicate) predicate.Predicate {
	return genericPredicate(func(v interface{}) bool {
		return p1(v) && p2.IsSatisfiedBy(v)
	})
}

// Or returns p1 || p2
func (p1 genericPredicate) Or(p2 predicate.Predicate) predicate.Predicate {
	return genericPredicate(func(v interface{}) bool {
		return p1(v) || p2.IsSatisfiedBy(v)
	})
}

// Negate returns Not(p1)
func (p1 genericPredicate) Negate() predicate.Predicate {
	return genericPredicate(func(v interface{}) bool {
		return !p1(v)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 genericPredicate) IsSatisfiedBy(v interface{}) bool {
	return p1(v)
}
