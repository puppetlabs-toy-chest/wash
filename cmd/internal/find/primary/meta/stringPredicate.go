package meta

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// StringPredicate => [^-].*
func parseStringPredicate(tokens []string) (Predicate, []string, error) {
	if len(tokens) == 0 || len(tokens[0]) == 0 {
		return nil, nil, errz.NewMatchError("expected a nonempty string")
	}
	token := tokens[0]
	if strings.Contains("-()!", string(token[0])) {
		// We impose this restriction to avoid conflicting with the
		// other primaries and any expression operators. Note that
		// this behavior's tested in parsePredicate's tests.
		var msg string
		if token[0] == '-' {
			msg = fmt.Sprintf("%v begins with a '-'", token)
		} else if len(token) == 1 {
			// token[0] is "(", ")", or "!"
			msg = fmt.Sprintf("%v is an expression operator", token)
		}
		return nil, nil, errz.NewMatchError(msg)
	}
	p := stringP(func(s string) bool {
		return s == token
	})
	return p, tokens[1:], nil
}

func stringP(p func(string) bool) *stringPredicate {
	return &stringPredicate{
		predicateBase: newPredicateBase(func(v interface{}) bool {
			strV, ok := v.(string)
			if !ok {
				return false
			}
			return p(strV)
		}),
		p: p,
	}
}

type stringPredicate struct {
	*predicateBase
	p func(string) bool
}

func (sp *stringPredicate) Negate() predicate.Predicate {
	nsp := stringP(func(s string) bool {
		return !sp.p(s)
	})
	nsp.negateSchemaP()
	return nsp
}
