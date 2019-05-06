package primary

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

//nolint
func newBooleanPrimary(val bool) *primary {
	return Parser.newPrimary([]string{fmt.Sprintf("-%v", val)}, func(tokens []string) (predicate.Entry, []string, error) {
		return func(e types.Entry) bool {
			return val
		}, tokens, nil
	})
}

// truePrimary => -true
//nolint
var truePrimary = newBooleanPrimary(true)

// falsePrimary => -false
//nolint
var falsePrimary = newBooleanPrimary(false)
