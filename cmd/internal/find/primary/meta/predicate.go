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

// Predicate is a wrapper to the predicate.Predicate interface.
// We'll need it later for the meta schema work.
type Predicate interface {
	predicate.Predicate
}

// predicateBase represents a `meta` primary predicate "base" class.
// Child classes must implement the Negate method.
type predicateBase func(interface{}) bool

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 predicateBase) IsSatisfiedBy(v interface{}) bool {
	return p1(v)
}

// genericPredicate represents a generic meta primary predicate that adheres
// to strict negation
type genericPredicate func(interface{}) bool

func (p1 genericPredicate) IsSatisfiedBy(v interface{}) bool {
	return p1(v)
}

func (p1 genericPredicate) Negate() predicate.Predicate {
	return genericPredicate((func(v interface{}) bool {
		return !p1(v)
	}))
}

// predicateAnd and predicateOr are necessary to strictly enforce De'Morgan's law.
// This is because child classes of predicateBase implement their own negate method.

type predicateAnd struct {
	predicateBase
	p1 Predicate
	p2 Predicate
}

func (op *predicateAnd) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	pp1 := p1.(Predicate)
	pp2 := p2.(Predicate)
	return &predicateAnd{
		p1: pp1,
		p2: pp2,
	}
}

func (op *predicateAnd) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) && op.p2.IsSatisfiedBy(v)
}

func (op *predicateAnd) Negate() predicate.Predicate {
	return (&predicateOr{}).Combine(op.p1.Negate(), op.p2.Negate())
}

type predicateOr struct {
	predicateBase
	p1 Predicate
	p2 Predicate
}

func (op *predicateOr) Combine(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
	pp1 := p1.(Predicate)
	pp2 := p2.(Predicate)
	return &predicateOr{
		p1: pp1,
		p2: pp2,
	}
}

func (op *predicateOr) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) || op.p2.IsSatisfiedBy(v)
}

func (op *predicateOr) Negate() predicate.Predicate {
	return (&predicateAnd{}).Combine(op.p1.Negate(), op.p2.Negate())
}
