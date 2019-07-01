package meta

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// A key consists of one or more characters that aren't
// ".", "[", or "]". The reason we have this limitation
// is so we can support key sequences, which includes
// nested keys like ".key1.key2", ".key1[?].key2", etc.
// key sequences aren't specified in the grammar because
// they make it harder to formalize the semantics.
var keyRegex = regexp.MustCompile(`^([^\.\[\]]+)`)

// ObjectPredicate => EmptyPredicate | ‘.’ Key Predicate
// Key             => keyRegex
func parseObjectPredicate(tokens []string) (predicate.Predicate, []string, error) {
	if p, tokens, err := parseEmptyPredicate(tokens); err == nil {
		return p, tokens, err
	}
	// EmptyPredicate did not match, so try '.' Key Predicate
	parseOAPredicate := predicate.ToParser(parseOAPredicate)
	return parseObjectP(
		tokens,
		parseOAPredicate,
		parseOAPredicate,
	)
}

// This helper's used by parseObjectPredicate and parseObjectExpression.
func parseObjectP(tokens []string, baseCaseParser, keySequenceParser predicate.Parser) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a key sequence")
	}
	tk := tokens[0]
	if tk[0] != '.' {
		return nil, nil, errz.NewMatchError("key sequences must begin with a '.'")
	}
	tk = tk[1:]
	loc := keyRegex.FindStringIndex(tk)
	if loc == nil {
		return nil, nil, fmt.Errorf("expected a key sequence after '.'")
	}
	key := tk[loc[0]:loc[1]]

	var p predicate.Predicate
	var err error
	if len(tk) == loc[1] {
		// tk is a single key, so it is of the form "key". This is the base case.
		tokens = tokens[1:]
		p, tokens, err = baseCaseParser.Parse(tokens)
	} else {
		// tk is a key sequence, so it is of the form "key1.key2" (or "key1[?]"). keyRegex
		// matched the "key1" part, while the ".key2"/"[?]" parts correspond to object/array
		// predicates. We can let keySequenceParser figure this info out for us by setting
		// tokens[0] to the regex's postmatch prior to passing it in.
		tokens[0] = tk[loc[1]:]
		p, tokens, err = keySequenceParser.Parse(tokens)
	}

	if err != nil {
		if errz.IsMatchError(err) {
			return nil, nil, fmt.Errorf("expected a predicate after %v", key)
		}
		return nil, nil, err
	}
	return objectP(key, p), tokens, nil
}

func objectP(key string, p predicate.Predicate) predicate.Predicate {
	return &objectPredicate{
		predicateBase: func(v interface{}) bool {
			mp, ok := v.(map[string]interface{})
			if !ok {
				return false
			}
			matchingKey := findMatchingKey(mp, key)
			if matchingKey == "" {
				// key doesn't exist in mp
				return false
			}
			return p.IsSatisfiedBy(mp[matchingKey])
		},
		key: key,
		p:   p,
	}
}

func findMatchingKey(mp map[string]interface{}, key string) string {
	upcasedKey := strings.ToUpper(key)
	for k := range mp {
		if strings.ToUpper(k) == upcasedKey {
			return k
		}
	}
	return ""
}

type objectPredicate struct {
	predicateBase
	key string
	p   predicate.Predicate
}

func (objP *objectPredicate) Negate() predicate.Predicate {
	return objectP(objP.key, objP.p.Negate())
}
