package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// The functionality here is tested in primary/meta_test.go

// Expression => EmptyPredicate | KeySequence PredicateExpression
//
// To simplify the parsing logic, we will change this to
//     Expression       => EmptyPredicate | ObjectExpression
//     ObjectExpression => '.' Key (PredicateExpression | OAExpression)
//     OAExpression     => ObjectExpression |
//                         ArrayExpression
//     ArrayExpression  => ‘[' ? ‘]’ (PredicateExpression | OAExpression) |
//                         ‘[' * ‘]’ (PredicateExpression | OAExpression) |
//                         ‘[' N ‘]’ (PredicateExpression | OAExpression) |
// 
// For ObjectExpression/ArrayExpression, assume their parsers are given a tokens
// array that's something like [<token>, <rest>...]. Then, if <token> does not
// contain a key sequence, <rest> will be parsed as a PredicateExpression. Otherwise,
// let <token> = <head>:<tail>, where <head> is the "'.' Key" part in ObjectExpression,
// or the "'[' ? ']'" part in ArrayExpression. Then, [<tail>, <rest>...] will be parsed
// as an OAExpression.
//
// NOTE: The rules in ^ are necessary so that something like
//     -meta .key1 .key2 -true -a .key3 -false
// is parsed as "the value of the 'key1' key in the entry's metadata is an object with 'key2'
// set to true AND is an object with 'key3' set to false". Otherwise, if we constructed the
// grammar as something like:
//     ObjectExpression => '.' Key Expr
//     Expr             => ObjectExpression    |
//                         ArrayExpression     |
//                         PredicateExpression
//     ArrayExpression  => '[' ? ']' Expr | ...
//
// (to mirror the symmetry of ObjectPredicate/ArrayPredicate), then our example would instead
// be parsed as "the value of the 'key1.key2' key in the entry's metadata is set to true AND it
// is an object with 'key3' set to false". We could get our example to parse correctly with the
// above grammar via something like "-meta .key1 \( .key2 -true -a .key3 -false \)", but that is
// annoying and unnecessary clutter.
//    
// Thus, the rules for PredicateExpression/OAExpression allow one to cleanly combine object/array
// predicates on entry metadata values without having to use a parentheses. 
func parseExpression(tokens []string) (predicate.Predicate, []string, error) {
	if p, tokens, err := parseEmptyPredicate(tokens); err == nil {
		return p, tokens, err
	}
	p, tokens, err := parseObjectExpression(tokens)
	if err != nil {
		if errz.IsMatchError(err) {
			// We expect an ObjectExpression, so treat any match errors
			// as syntax errors.
			err = fmt.Errorf(err.Error())
		}
		return nil, nil, err
	}
	return p, tokens, nil
}

func parseObjectExpression(tokens []string) (predicate.Predicate, []string, error) {
	return parseObjectP(
		tokens,
		predicate.ToParser(parsePredicateExpression),
		predicate.ToParser(parseOAExpression),
	)
}

func parseOAExpression(tokens []string) (predicate.Predicate, []string, error) {
	cp := &predicate.CompositeParser{
		// This error message should never be used. The following comment explains why.
		MatchErrMsg: "",
		Parsers: []predicate.Parser{
			predicate.ToParser(parseObjectExpression),
			predicate.ToParser(parseArrayExpression),
		},
	}
	p, tokens, err := cp.Parse(tokens)
	if err != nil {
		if errz.IsMatchError(err) {
			// We should never hit this code-path because parseOAExpression
			// will only be called by parseObjectExpression/parseArrayExpression
			// when either method receives a key sequence. Key sequences will always
			// match parseObjectExpression/parseArrayExpression.
			msg := fmt.Sprintf(
				"meta.parseOAExpression: predicate.CompositeParser#Parse returned a match error on a key sequence. Input: %v",
				tokens,
			)
			panic(msg)
		}
		return nil, nil, err
	}
	return p, tokens, nil
}

func parseArrayExpression(tokens []string) (predicate.Predicate, []string, error) {
	return parseArrayP(
		tokens,
		predicate.ToParser(parsePredicateExpression),
		predicate.ToParser(parseOAExpression),
	)
}