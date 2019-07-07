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
		return nullP(), tokens[1:], nil
	case "-exists":
		// ExistsPredicate
		return existsP(), tokens[1:], nil
	case "-true":
		// BooleanPredicate
		return trueP(), tokens[1:], nil
	case "-false":
		// BooleanPredicate
		return falseP(), tokens[1:], nil
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

func nullP() Predicate {
	return genericP(func(v interface{}) bool {
		return v == nil
	})
}

func existsP() Predicate {
	gp := genericP(func(v interface{}) bool {
		return v != nil
	})
	gp.SchemaP = newExistsPredicateSchemaP(false)
	return gp
}

type existsPredicateSchemaP struct {
	ks      keySequence
	negated bool
}

func newExistsPredicateSchemaP(negated bool) *existsPredicateSchemaP {
	return &existsPredicateSchemaP{
		ks:      (keySequence{}).CheckExistence(),
		negated: negated,
	}
}

func (p *existsPredicateSchemaP) IsSatisfiedBy(v interface{}) bool {
	s, ok := v.(schema)
	if !ok {
		return false
	}
	result := s.IsValidKeySequence(p.ks)
	if p.negated {
		// ".key ! -exists" will return false if m['key'] does not exist.
		// This implies that the corresponding schemaP should also return
		// false, i.e. that we adhere to strict negation.
		result = !result
	}
	return result
}

func (p *existsPredicateSchemaP) Negate() predicate.Predicate {
	return newExistsPredicateSchemaP(!p.negated)
}

func (p *existsPredicateSchemaP) updateKS(updateFunc func(keySequence) keySequence) {
	p.ks = updateFunc(p.ks)
}

// Note that we can't set trueP/falseP to variables because their schema
// predicates need to be re-created each time they're parsed.
func trueP() *booleanPredicate {
	return booleanP(true)
}

func falseP() *booleanPredicate {
	return booleanP(false)
}

func booleanP(value bool) *booleanPredicate {
	return &booleanPredicate{
		predicateBase: newPredicateBase(func(v interface{}) bool {
			bv, ok := v.(bool)
			if !ok {
				return false
			}
			return bv == value
		}),
		value: value,
	}
}

type booleanPredicate struct {
	*predicateBase
	value bool
}

func (bp *booleanPredicate) Negate() predicate.Predicate {
	nbp := booleanP(!bp.value)
	nbp.negateSchemaP()
	return nbp
}
