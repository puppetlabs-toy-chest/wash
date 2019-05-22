package primary

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

//nolint
func newBooleanPrimary(val bool) *Primary {
	return Parser.add(&Primary{
		Description: fmt.Sprintf("Always returns %v", val),
		name: fmt.Sprintf("%v", val),
		parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
			return func(e types.Entry) bool {
				return val
			}, tokens, nil
		},
	})
}

// True is the true primary
//
// truePrimary => -true
//nolint
var True = newBooleanPrimary(true)

// False is the false primary
//
// falsePrimary => -false
//nolint
var False = newBooleanPrimary(false)
