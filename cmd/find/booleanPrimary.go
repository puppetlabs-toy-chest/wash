package cmdfind

import (
	"fmt"

	apitypes "github.com/puppetlabs/wash/api/types"
)

//nolint
func newBooleanPrimary(val bool) *atom {
	return newAtom([]string{fmt.Sprintf("-%v", val)}, func(tokens []string) (Predicate, []string, error) {
		return func(e *apitypes.ListEntry) bool {
			return val
		}, tokens[1:], nil
	})
}

// truePrimary => -true
//nolint
var truePrimary = newBooleanPrimary(true)

// falsePrimary => -false
//nolint
var falsePrimary = newBooleanPrimary(false)
