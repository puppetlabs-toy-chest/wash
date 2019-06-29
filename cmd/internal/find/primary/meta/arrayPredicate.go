package meta

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// ArrayPredicate => EmptyPredicate      |
//                   ‘[' ? ‘]’ Predicate |
//                   ‘[' * ‘]’ Predicate |
//                   ‘[' N ‘]’ Predicate |
func parseArrayPredicate(tokens []string) (predicate.Predicate, []string, error) {
	if p, tokens, err := parseEmptyPredicate(tokens); err == nil {
		return p, tokens, err
	}
	// EmptyPredicate did not match
	parseOAPredicate := predicate.ToParser(parseOAPredicate)
	return parseArrayP(
		tokens,
		parseOAPredicate,
		parseOAPredicate,
	)
}

// This helper's used by parseArrayPredicate and parseArrayExpression
func parseArrayP(tokens []string, baseCaseParser, keySequenceParser predicate.Parser) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected an opening '['")
	}
	rawToken := tokens[0]
	// For simplicity, we require rawToken to be complete. Thus, if
	// tokens is something like ["[", "]"] or ["[", "*", "]"], then
	// parseArrayPredicateType will return an error.
	ptype, token, err := parseArrayPredicateType(rawToken)
	if err != nil {
		return nil, nil, err
	}

	var p predicate.Predicate
	if len(token) == 0 {
		tokens = tokens[1:]
		p, tokens, err = baseCaseParser.Parse(tokens)
	} else {
		// token may be part of a key sequence (e.g. something like
		// [?].key2 or [?][?])
		if token[0] != '.' && token[0] != '[' {
			// Returning this error avoids weird cases like "[?]-true", which would
			// otherwise be parsed as an array predicate on a Boolean predicate. For
			// that case, the token being compared here would be "-true".
			return nil, nil, fmt.Errorf("expected a '.' or '[' after ']' but got %v instead", token)
		}
		tokens[0] = token
		p, tokens, err = keySequenceParser.Parse(tokens)
	}

	if err != nil {
		if errz.IsMatchError(err) {
			parsedToken := rawToken[0 : len(rawToken)-len(token)]
			return nil, nil, fmt.Errorf("expected a predicate after %v", parsedToken)
		}
		return nil, nil, err
	}
	return arrayP(ptype, p), tokens, nil
}

type arrayPredicateType struct {
	t byte
	n uint
}

// parseArrayPredicateType parses the array predicate type from the
// given token. It returns the array predicate's type and the
// remaining bit of the token.
func parseArrayPredicateType(token string) (arrayPredicateType, string, error) {
	ptype := arrayPredicateType{}
	if len(token) == 0 || token[0] != '[' {
		msg := "expected an opening '['"
		if token[0] == ']' {
			return ptype, "", fmt.Errorf(msg)
		}
		return ptype, "", errz.NewMatchError(msg)
	}
	token = token[1:]
	endIx := strings.Index(token, "]")
	if endIx < 0 {
		return ptype, "", fmt.Errorf("expected a closing ']'")
	}
	if endIx == 0 {
		return ptype, "", fmt.Errorf("expected a '*', '?', or an array index inside '[]'")
	}
	if token[0] == '*' || token[0] == '?' {
		if endIx > 1 {
			// Handles input like [*123] or [?123]
			return ptype, "", fmt.Errorf("expected a closing ']' after '%v'", string(token[0]))
		}
		if token[0] == '*' {
			// a => any
			ptype.t = 'a'
		} else {
			// s => some
			ptype.t = 's'
		}
	} else {
		n, err := strconv.ParseUint(token[0:endIx], 10, 32)
		if err != nil {
			return ptype, "", fmt.Errorf("expected an array index inside '[]'")
		}
		// n => nth
		ptype.t = 'n'
		ptype.n = uint(n)
	}
	return ptype, token[endIx+1:], nil
}

func arrayP(ptype arrayPredicateType, p predicate.Predicate) predicate.Predicate {
	arryP := &arrayPredicate{
		ptype: ptype,
		p:     p,
	}
	switch ptype.t {
	case 's':
		arryP.predicateBase = toArrayP(func(vs []interface{}) bool {
			for _, v := range vs {
				if p.IsSatisfiedBy(v) {
					return true
				}
			}
			// p(v) returned false for all v in vs, so return
			// false
			return false
		})
	case 'a':
		arryP.predicateBase = toArrayP(func(vs []interface{}) bool {
			for _, v := range vs {
				if !p.IsSatisfiedBy(v) {
					return false
				}
			}
			// p(v) returned true for all v in vs, so return true
			return true
		})
	case 'n':
		n := ptype.n
		arryP.predicateBase = toArrayP(func(vs []interface{}) bool {
			if n >= uint(len(vs)) {
				return false
			}
			return p.IsSatisfiedBy(vs[n])
		})
	default:
		msg := fmt.Sprintf("meta.arrayP called with an unkown ptype %v", ptype.t)
		panic(msg)
	}
	return arryP
}

// toArrayP is a helper for arrayP that's meant to reduce
// the boilerplate type validation.
func toArrayP(p func([]interface{}) bool) predicateBase {
	return predicateBase(func(v interface{}) bool {
		arrayV, ok := v.([]interface{})
		if !ok {
			return false
		}
		return p(arrayV)
	})
}

type arrayPredicate struct {
	predicateBase
	ptype arrayPredicateType
	p     predicate.Predicate
}

func (arryP *arrayPredicate) Negate() predicate.Predicate {
	switch t := arryP.ptype.t; t {
	case 's':
		// Not("Some element satisfies p") => "All elements satisfy Not(p)"
		ptype := arrayPredicateType{}
		ptype.t = 'a'
		return arrayP(ptype, arryP.p.Negate())
	case 'a':
		// Not("All elements satisfy p") => "Some element satisfies Not(p)"
		ptype := arrayPredicateType{}
		ptype.t = 's'
		return arrayP(ptype, arryP.p.Negate())
	case 'n':
		// Not("Nth element satisfies p") => "Nth element satisfies Not(p)"
		return arrayP(arryP.ptype, arryP.p.Negate())
	default:
		msg := fmt.Sprintf("meta.arrayPredicate contains an unknown ptype %v", t)
		panic(msg)
	}
}
