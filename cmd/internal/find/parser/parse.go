package parser

import (
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// Result represents the result of parsing `wash find`'s
// arguments.
type Result struct {
	Predicate types.Predicate
}

// Parse parses `wash find`'s arguments, returning the result.
func Parse(args []string) (Result, error) {
	r := Result{}
	p, err := parseExpression(args)
	if err != nil {
		return r, err
	}
	r.Predicate = p
	return r, nil
}
