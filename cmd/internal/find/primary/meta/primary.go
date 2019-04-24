// Package meta contains all the parsing logic for the `meta` primary
package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/grammar"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

/*
Primary            => (-meta|-m) ObjectPredicate

ObjectPredicate    => EmptyPredicate | ‘.’ Key Predicate
EmptyPredicate     => -empty
Key                => [ ^.[ ] ]+ (i.e. one or more cs that aren't ".", "[", or "]")

Predicate          => ObjectPredicate     |
                      ArrayPredicate      |
                      PrimitivePredicate

ArrayPredicate     => EmptyPredicate      |
                      ‘[' ? ‘]’ Predicate |
                      ‘[' * ‘]’ Predicate |
                      ‘[' N ‘]’ Predicate |

PrimitivePredicate => NullPredicate       |
                      ExistsPredicate     |
                      BooleanPredicate    |
                      NumericPredicate    |
                      TimePredicate       |
                      StringPredicate

NullPredicate      => -null
ExistsPredicate    => -exists
BooleanPredicate   => -true | -false

NumericPredicate   => (+|-)? Number
Number             => N | '{' N '}' | numeric.SizeRegex

TimePredicate      => (+|-)? Duration
Duration           => numeric.DurationRegex | '{' numeric.DurationRegex '}'

StringPredicate    => [^-].*

N                  => \d+ (i.e. some number > 0)
*/
//nolint
var Primary = grammar.NewAtom([]string{"-meta", "-m"}, func(tokens []string) (types.Predicate, []string, error) {
	// tokens[0] contains the (-meta|-m) part
	token := tokens[0]
	p, tokens, err := parseObjectPredicate(tokens[1:])
	if err != nil {
		return nil, nil, fmt.Errorf("%v: %v", token, err)
	}
	return func(e types.Entry) bool {
		mp := map[string]interface{}(e.Attributes.Meta())
		return p(mp)
	}, tokens, nil
})
