package meta

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

// A key consists of one or more characters that aren't
// ".", "[", or "]". The reason we have this limitation
// is so we can support key sequences, which includes
// nested keys like ".key1.key2", ".key1[].key2", etc.
// key sequences aren't specified in the grammar because
// they make it harder to formalize the semantics.
var keyRegex = regexp.MustCompile(`^([^\.\[\]]+)`)

// ObjectPredicate => EmptyPredicate | ‘.’ Key Predicate
// Key             => keyRegex
func parseObjectPredicate(tokens []string) (predicate, []string, error) {
	if p, tokens, err := parseEmptyPredicate(tokens); err == nil {
		return p, tokens, err
	}
	// EmptyPredicate did not match, so try '.' Key Predicate
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
	if len(tk) == loc[1] {
		// tk is not a key sequence, so we can shift tokens
		tokens = tokens[1:]
	} else {
		// tk is a key sequence, so shift it to the regex's postmatch
		tokens[0] = tk[loc[1]:]
	}
	p, tokens, err := parsePredicate(tokens)
	if err != nil {
		if errz.IsMatchError(err) {
			return nil, nil, fmt.Errorf("expected a predicate after %v", key)
		}
		return nil, nil, err
	}
	return objectP(key, p), tokens, nil
}

func objectP(key string, p predicate) predicate {
	return func(v interface{}) bool {
		mp, ok := v.(map[string]interface{})
		if !ok {
			return false
		}
		matchingKey := findMatchingKey(mp, key)
		if matchingKey == "" {
			// key doesn't exist in mp
			return false
		}
		return p(mp[matchingKey])
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
