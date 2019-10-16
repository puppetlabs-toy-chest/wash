package meta

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// ObjectPredicate => EmptyPredicate | ‘.’ Key Predicate
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
	key, rem, err := parseKey(tk)
	if err != nil {
		return nil, nil, err
	}
	var p predicate.Predicate
	if len(rem) <= 0 {
		// tk is a single key, so it is of the form "key". This is the base case.
		tokens = tokens[1:]
		p, tokens, err = baseCaseParser.Parse(tokens)
	} else {
		// tk is a key sequence, so it is of the form "key1.key2" (or "key1[?]"). keyRegex
		// matched the "key1" part, while the ".key2"/"[?]" parts correspond to object/array
		// predicates. We can let keySequenceParser figure this info out for us by setting
		// tokens[0] to the remaining part of tk
		tokens[0] = rem
		p, tokens, err = keySequenceParser.Parse(tokens)
	}
	if err != nil {
		if errz.IsMatchError(err) {
			err = fmt.Errorf("expected a predicate after %v", key)
		}
	}
	return objectP(key, p), tokens, err
}

// A key consists of one or more characters that aren't
// ".", "[", or "]". The reason we have this limitation
// is so we can support key sequences, which includes
// nested keys like ".key1.key2", ".key1[?].key2", etc.
// key sequences aren't specified in the grammar because
// they make it harder to formalize the semantics.
//
// NOTE: Users can still specify ".", "[", or "]" by escaping
// them with a backslash "\". For example, '.com\.docker\.compose'
// would be parsed as the key "com.docker.compose".
func parseKey(tk string) (string, string, error) {
	isTerminatingChar := func(char byte) bool {
		return char == '.' || char == '[' || char == ']'
	}
	isEscapableChar := func(char byte) bool {
		return isTerminatingChar(char) || char == '\\'
	}

	if len(tk) <= 0 || tk[0] != '.' {
		return "", "", errz.NewMatchError("key sequences must begin with a '.'")
	}
	key := ""
	rem := tk[1:]
	for {
		if len(rem) <= 0 {
			break
		}
		if isTerminatingChar(rem[0]) {
			break
		}
		if rem[0] == '\\' {
			// Lookahead and check for an escapable character
			if len(rem) <= 1 || !isEscapableChar(rem[1]) {
				return "", "", fmt.Errorf("no escapable character specified after the '\\'")
			}
			rem = rem[1:]
		}
		key += string(rem[0])
		rem = rem[1:]
	}
	if len(key) <= 0 {
		return "", "", fmt.Errorf("expected a key sequence after '.'")
	}
	return key, rem, nil
}

func objectP(key string, p predicate.Predicate) Predicate {
	if p == nil {
		return nil
	}
	objP := &objectPredicate{
		predicateBase: newPredicateBase(func(v interface{}) bool {
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
		}),
		key: key,
		p:   p,
	}
	objP.SchemaP = p.(Predicate).schemaP()
	objP.SchemaP.updateKS(func(ks keySequence) keySequence {
		return ks.AddObject(key)
	})
	return objP
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
	*predicateBase
	key string
	p   predicate.Predicate
}

func (objP *objectPredicate) Negate() predicate.Predicate {
	// ! .key p == .key ! p
	//
	// Note that these semantics also hold for schemaP negation.
	return objectP(objP.key, objP.p.Negate())
}
