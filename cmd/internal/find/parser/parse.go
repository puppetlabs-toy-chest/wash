package parser

import (
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// Result represents the result of parsing `wash find`'s
// arguments.
type Result struct {
	Path      string
	Predicate types.Predicate
}

/*
Parse parses `wash find`'s arguments, returning the result.
`wash find`'s arguments are specified as "[path] [expression]"
*/
func Parse(args []string) (Result, error) {
	r := Result{}
	r.Path, args = parsePath(args)
	p, err := parseExpression(args)
	if err != nil {
		return r, err
	}
	r.Predicate = p
	return r, nil
}
