package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
)

// NumericPredicate => (+|-)? Number
// Number           => N | '{' N '}' | numeric.SizeRegex
func parseNumericPredicate(tokens []string) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a +, -, or a digit")
	}
	token := tokens[0]
	p, _, err := numeric.ParsePredicate(
		token,
		numeric.ParsePositiveInt,
		numeric.Bracket(numeric.Negate(numeric.ParsePositiveInt)),
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
	return numericP(p), tokens[1:], nil
}

func numericP(p numeric.Predicate) predicate.Predicate {
	return &numericPredicate{
		predicateBase: func(v interface{}) bool {
			floatV, ok := v.(float64)
			if !ok {
				return false
			}
			return p(int64(floatV))
		},
		p: p,
	}
}

type numericPredicate struct {
	predicateBase
	p numeric.Predicate
}

func (np *numericPredicate) Negate() predicate.Predicate {
	return numericP(np.p.Negate().(numeric.Predicate))
}
