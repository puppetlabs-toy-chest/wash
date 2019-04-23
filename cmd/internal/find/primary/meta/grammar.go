package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

// predicateParser represents a type that parses `meta` primary predicates.
type predicateParser func(tokens []string) (predicate, []string, error)

// try attempts to parse the given tokens by trying each of the passed-in parsers.
// It returns the result of the first parser that parses the given predicate.
// If none of the parsers parse the given predicate, then an error is returned.
// Callers should construct their own error message off this returned error.
func try(tokens []string, parsers ...predicateParser) (predicate, []string, error) {
	for _, parser := range parsers {
		p, tokens, err := parser(tokens)
		if err == nil {
			return p, tokens, err
		}
		if errz.IsMatchError(err) {
			// Try the next parser
			continue
		}
		// The given parser matched the input, but returned a parse error. Thus,
		// we return the error.
		return nil, nil, err
	}
	return nil, nil, errz.NewMatchError("none of the parsers matched the provided input")
}
