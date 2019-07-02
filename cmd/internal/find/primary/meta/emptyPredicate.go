package meta

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// emptyPredicate => -empty
func parseEmptyPredicate(tokens []string) (Predicate, []string, error) {
	if len(tokens) == 0 || tokens[0] != "-empty" {
		return nil, nil, errz.NewMatchError("expected '-empty'")
	}
	return emptyP(false), tokens[1:], nil
}

func emptyP(negated bool) Predicate {
	ep := &emptyPredicate{
		predicateBase: newPredicateBase(func(v interface{}) bool {
			switch t := v.(type) {
			case map[string]interface{}:
				if negated {
					return len(t) > 0
				}
				return len(t) == 0
			case []interface{}:
				if negated {
					return len(t) > 0
				}
				return len(t) == 0
			default:
				return false
			}
		}),
		negated: negated,
	}
	// An empty predicate's schemaP returns true iff the value's
	// an empty array OR an empty object.
	ep.SchemaP = &emptyPredicateSchemaP{
		schemaPOr: newSchemaPOr(
			newObjectValueSchemaP(),
			newArrayValueSchemaP(),
		),
	}
	return ep
}

type emptyPredicate struct {
	*predicateBase
	negated bool
}

func (p *emptyPredicate) Negate() predicate.Predicate {
	return emptyP(!p.negated)
}

type emptyPredicateSchemaP struct {
	*schemaPOr
}

func (p1 *emptyPredicateSchemaP) Negate() predicate.Predicate {
	// The empty predicate's negation still expects an empty object/array
	// for its schemaP. Thus, we return p1 here.
	return p1
}
