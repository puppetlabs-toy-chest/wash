package primary

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

//nolint
func newBooleanPrimary(val bool) *Primary {
	return Parser.add(&Primary{
		Description: fmt.Sprintf("Always returns %v", val),
		name:        fmt.Sprintf("%v", val),
		parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
			p := types.ToEntryP(func(e types.Entry) bool {
				return val
			})
			p.SetSchemaP(types.ToEntrySchemaP(func(s *types.EntrySchema) bool {
				return val
			}))
			return p, tokens, nil
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
