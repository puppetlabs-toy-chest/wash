package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

/*
PrimitivePredicate => NullPredicate       |
                      ExistsPredicate     |
                      BooleanPredicate    |
                      NumericPredicate    |
                      TimePredicate       |
                      StringPredicate
*/
func parsePrimitivePredicate(tokens []string) (predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a primitive predicate")
	}
	switch token := tokens[0]; token {
	case "-null":
		// NullPredicate
		return nullP, tokens[1:], nil
	case "-exists":
		// ExistsPredicate
		return existsP, tokens[1:], nil
	case "-true":
		// BooleanPredicate
		return trueP, tokens[1:], nil
	case "-false":
		// BooleanPredicate
		return falseP, tokens[1:], nil
	}
	p, tokens, err := try(
		tokens,
		parseNumericPredicate,
		parseTimePredicate,
		// We place parseStringPredicate last because it's meant to be a fallback
		// in case the previous parsers return match errors.
		parseStringPredicate,
	)
	if err != nil {
		if errz.IsMatchError(err) {
			return nil, nil, errz.NewMatchError("expected a primitive predicate")
		}
		return nil, nil, err
	}
	return p, tokens, nil
}

func nullP(v interface{}) bool {
	return v == nil
}

func existsP(v interface{}) bool {
	return v != nil
}

var trueP = makeBooleanPredicate(true)
var falseP = makeBooleanPredicate(false)

func makeBooleanPredicate(expectedVal bool) predicate {
	return func(v interface{}) bool {
		bv, ok := v.(bool)
		if !ok {
			return false
		}
		return bv == expectedVal
	}
}
