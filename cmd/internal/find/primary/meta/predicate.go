package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// Predicate represents a meta primary predicate.
type Predicate func(interface{}) bool

// And returns p1 && p2
func (p1 Predicate) And(p2 predicate.Predicate) predicate.Predicate {
	return Predicate(func(v interface{}) bool {
		return p1(v) && (p2.(Predicate))(v)
	})
}

// Or returns p1 || p2
func (p1 Predicate) Or(p2 predicate.Predicate) predicate.Predicate {
	return Predicate(func(v interface{}) bool {
		return p1(v) || (p2.(Predicate))(v)
	})
}

// Negate returns Not(p1)
func (p1 Predicate) Negate() predicate.Predicate {
	return Predicate(func(v interface{}) bool {
		return !p1(v)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 Predicate) IsSatisfiedBy(v interface{}) bool {
	return p1(v)
}

/*
Predicate => ObjectPredicate |
             ArrayPredicate  |
             PrimitivePredicate
*/
func parsePredicate(tokens []string) (Predicate, []string, error) {
	cp := &predicate.CompositeParser{
		ErrMsg: "expected either a primitive, object, or array predicate",
		Parsers: []predicate.Parser{
			predicateParser(parseObjectPredicate),
			predicateParser(parseArrayPredicate),
			predicateParser(parsePrimitivePredicate),
		},
	}
	p, tokens, err := cp.Parse(tokens)
	if err != nil {
		return nil, nil, err
	}
	return p.(Predicate), tokens, err
}


type predicateParser func(tokens []string) (Predicate, []string, error)

// Parse parses a meta primary predicate from the given input.
func (parser predicateParser) Parse(tokens []string) (predicate.Predicate, []string, error) {
	return parser(tokens)
}

