package meta

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

// ArrayPredicate => EmptyPredicate      |
//                   ‘[' ‘]’ Predicate   |
//                   ‘[' * ‘]’ Predicate |
//                   ‘[' N ‘]’ Predicate |
func parseArrayPredicate(tokens []string) (predicate, []string, error) {
	if p, tokens, err := parseEmptyPredicate(tokens); err == nil {
		return p, tokens, err
	}
	// EmptyPredicate did not match
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
	if len(token) == 0 {
		tokens = tokens[1:]
	} else {
		// token may be part of a key sequence (e.g. something like
		// [].key2 or [][])
		if token[0] != '.' && token[0] != '[' {
			// Returning this error avoids weird cases like "[]-true", which would
			// otherwise be parsed as an array predicate on a Boolean predicate. For
			// that case, the token being compared here would be "-true".
			return nil, nil, fmt.Errorf("expected a '.' or '[' after ']' but got %v instead", token)
		}
		tokens[0] = token
	}
	p, tokens, err := parsePredicate(tokens)
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
		// s => some
		ptype.t = 's'
	} else if token[0] == '*' {
		if endIx > 1 {
			return ptype, "", fmt.Errorf("expected a closing ']' after '*'")
		}
		// a => any
		ptype.t = 'a'
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

func arrayP(ptype arrayPredicateType, p predicate) predicate {
	switch ptype.t {
	case 's':
		return toArrayP(func(vs []interface{}) bool {
			for _, v := range vs {
				if p(v) {
					return true
				}
			}
			// p(v) returned false for all v in vs, so return
			// false
			return false
		})
	case 'a':
		return toArrayP(func(vs []interface{}) bool {
			for _, v := range vs {
				if !p(v) {
					return false
				}
			}
			// p(v) returned true for all v in vs, so return true
			return true
		})
	case 'n':
		n := ptype.n
		return toArrayP(func(vs []interface{}) bool {
			if n >= uint(len(vs)) {
				return false
			}
			return p(vs[n])
		})
	default:
		msg := fmt.Sprintf("meta.arrayP called with an unkown ptype %v", ptype.t)
		panic(msg)
	}
}

// toArrayP is a helper for arrayP that's meant to reduce
// the boilerplate type validation.
func toArrayP(p func([]interface{}) bool) predicate {
	return func(v interface{}) bool {
		arrayV, ok := v.([]interface{})
		if !ok {
			return false
		}
		return p(arrayV)
	}
}
