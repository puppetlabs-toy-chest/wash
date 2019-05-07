package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

/*
PrimitivePredicate => NullPredicate       |
                      ExistsPredicate     |
                      BooleanPredicate    |
                      NumericPredicate    |
                      TimePredicate       |
                      StringPredicate
*/
func parsePrimitivePredicate(tokens []string) (predicate.Predicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a primitive predicate")
	}
	switch token := tokens[0]; token {
	case "-null":
		// NullPredicate
		return genericPredicate(nullP), tokens[1:], nil
	case "-exists":
		// ExistsPredicate
		return genericPredicate(existsP), tokens[1:], nil
	case "-true":
		// BooleanPredicate
		return trueP, tokens[1:], nil
	case "-false":
		// BooleanPredicate
		return falseP, tokens[1:], nil
	}
	// We have either a NumericPredicate, a TimePredicate, or a StringPredicate.

	// Use tks to avoid overwriting tokens
	p, tks, err := parseNumericPredicate(tokens)
	if err == nil {
		return p, tks, nil
	}
	// parseNumericPredicate returned an error. Store the error, then
	// try parseTimePredicate. If parseTimePredicate returns a syntax
	// error, return it. Otherwise, check if numericPErr was a syntax
	// error. If so, return it. We need this complicated logic to handle
	// parsing "negative" time predicates like +{2h}, because parseNumericPredicate
	// will return syntax errors for those values even though they're valid
	// primitive predicates.
	numericPErr := err
	p, tks, err = parseTimePredicate(tokens)
	if err == nil {
		return p, tks, nil
	}
	if !errz.IsMatchError(err) {
		return nil, nil, err
	}
	if !errz.IsMatchError(numericPErr) {
		// numericPErr was a syntax error, so return it.
		return nil, nil, numericPErr
	}

	// Fallback to parseStringPredicate
	p, tks, err = parseStringPredicate(tokens)
	if err == nil {
		return p, tks, nil
	}
	if errz.IsMatchError(err) {
		err = errz.NewMatchError("expected a primitive predicate")
	}
	return nil, nil, err
}

func nullP(v interface{}) bool {
	return v == nil
}

func existsP(v interface{}) bool {
	return v != nil
}

var trueP = booleanP(true)
var falseP = booleanP(false)

func booleanP(value bool) predicate.Predicate {
	return &booleanPredicate{
		genericPredicate: func(v interface{}) bool {
			bv, ok := v.(bool)
			if !ok {
				return false
			}
			return bv == value
		},
		value: value,
	}
}

type booleanPredicate struct {
	genericPredicate
	value bool
}

func (bp *booleanPredicate) Negate() predicate.Predicate {
	return booleanP(!bp.value)
}
