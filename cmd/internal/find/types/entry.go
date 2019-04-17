package types

import apitypes "github.com/puppetlabs/wash/api/types"

// Entry represents an Entry as interpreted by `wash find`
type Entry struct {
	apitypes.Entry
	NormalizedPath string
}
