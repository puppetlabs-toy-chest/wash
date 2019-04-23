package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
)

// NumericPredicate => (+|-)? Number
// Number           => N | '{' N '}' | numeric.SizeRegex
func parseNumericPredicate(tokens []string) (predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a +, -, or a digit")
	}
	token := tokens[0]
	numericP, _, err := numeric.ParsePredicate(
		token,
		numeric.ParsePositiveInt,
		numeric.Negate(numeric.ParsePositiveInt),
		numeric.ParseSize,
	)
	if err != nil {
		if errz.IsMatchError(err) {
			msg := fmt.Sprintf("expected a number but got %v", token)
			return nil, nil, errz.NewMatchError(msg)
		}
		// err is a parse error, so return it.
		return nil, nil, err
	}
	p := func(v interface{}) bool {
		floatV, ok := v.(float64)
		if !ok {
			return false
		}
		return numericP(int64(floatV))
	}
	return p, tokens[1:], nil
}
