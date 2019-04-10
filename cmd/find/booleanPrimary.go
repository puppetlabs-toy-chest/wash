package cmdfind

import (
	"fmt"

	apitypes "github.com/puppetlabs/wash/api/types"
)

func newBooleanPrimary(val bool) *atom {
	return newAtom([]string{fmt.Sprintf("-%v", val)}, func(tokens []string) (Predicate, []string, error) {
		return func(e *apitypes.ListEntry) bool {
			return val
		}, tokens[1:], nil
	})
}

// truePrimary => -true
var truePrimary = newBooleanPrimary(true)

// falsePrimary => -false
var falsePrimary = newBooleanPrimary(false)
