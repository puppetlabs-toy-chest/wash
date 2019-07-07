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
// It is useful for extracing the schemaP without having to rely
// on a giant type switch
type Predicate interface {
	predicate.Predicate
	schemaP() schemaPredicate
}

// predicateBase represents a `meta` primary predicate "base" class.
// Child classes must implement the Negate method. They must also
// remember to negate the schemaP (where appropriate).
type predicateBase struct {
	P       func(interface{}) bool
	SchemaP schemaPredicate
}

func newPredicateBase(p func(interface{}) bool) *predicateBase {
	return &predicateBase{
		P: p,
		// This is the common case.
		SchemaP: newPrimitiveValueSchemaP(),
	}
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 *predicateBase) IsSatisfiedBy(v interface{}) bool {
	return p1.P(v)
}

// Helper that negates p1's schemaP.
func (p1 *predicateBase) negateSchemaP() {
	p1.SchemaP = p1.SchemaP.Negate().(schemaPredicate)
}

func (p1 *predicateBase) schemaP() schemaPredicate {
	return p1.SchemaP
}

// genericPredicate represents a generic meta primary predicate that adheres
// to strict negation
type genericPredicate struct {
	*predicateBase
}

func genericP(p func(interface{}) bool) *genericPredicate {
	return &genericPredicate{
		predicateBase: newPredicateBase(p),
	}
}

func (p1 *genericPredicate) Negate() predicate.Predicate {
	gp := genericP(func(v interface{}) bool {
		return !p1.P(v)
	})
	gp.SchemaP = p1.SchemaP
	gp.negateSchemaP()
	return gp
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
	andp := &predicateAnd{
		p1: pp1,
		p2: pp2,
	}
	andp.SchemaP = newSchemaPAnd(pp1.schemaP(), pp2.schemaP())
	return andp
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
	orp := &predicateOr{
		p1: pp1,
		p2: pp2,
	}
	orp.SchemaP = newSchemaPOr(pp1.schemaP(), pp2.schemaP())
	return orp
}

func (op *predicateOr) IsSatisfiedBy(v interface{}) bool {
	return op.p1.IsSatisfiedBy(v) || op.p2.IsSatisfiedBy(v)
}

func (op *predicateOr) Negate() predicate.Predicate {
	return (&predicateAnd{}).Combine(op.p1.Negate(), op.p2.Negate())
}
