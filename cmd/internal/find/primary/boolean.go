package primary

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

//nolint
func newBooleanPrimary(val bool) *primary {
	return Parser.add(&primary{
		tokens: []string{fmt.Sprintf("-%v", val)},
		parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
			return func(e types.Entry) bool {
				return val
			}, tokens, nil
		},
	})
}

// truePrimary => -true
//nolint
var truePrimary = newBooleanPrimary(true)

// falsePrimary => -false
//nolint
var falsePrimary = newBooleanPrimary(false)
