package primary

import (
	"fmt"
	"math"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// sizePrimary => -size (+|-)?(\d+ | numeric.SizeRegex)
//
// Example inputs:
//   -size 2   (true if the entry's size in 512-byte blocks, rounded up, is 2)
//   -size +2  (true if the entry's size in 512-byte blocks, rounded up, is greater than 2)
//   -size -2  (true if the entry's size in 512-byte blocks, rounded up, is less than 2)
//   -size +1k (true if the entry's size is greater than 1 kibibyte (1024 bytes))
//
//nolint
var sizePrimary = Parser.newPrimary([]string{"-size"}, func(tokens []string) (types.EntryPredicate, []string, error) {
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("requires additional arguments")
	}
	numericP, parserID, err := numeric.ParsePredicate(
		tokens[0],
		numeric.ParsePositiveInt,
		numeric.ParseSize,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("%v: illegal size value", tokens[0])
	}

	p := func(e types.Entry) bool {
		if !e.Attributes.HasSize() {
			return false
		}
		size := int64(e.Attributes.Size())
		if parserID == 0 {
			// n was an integer, so convert the size to the # of 512-byte blocks (rounded up)
			size = int64(math.Ceil(float64(size) / 512.0))
		}
		return numericP(size)
	}
	return p, tokens[1:], nil
})
