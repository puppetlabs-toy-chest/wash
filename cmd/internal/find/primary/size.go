package primary

import (
	"fmt"
	"math"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// sizePrimary => -size (+|-)?(\d+ | numeric.SizeRegex)
//
//nolint
var sizePrimary = Parser.add(&Primary{
	Description: "Returns true if the entry's size attribute satisfies the given size predicate",
	DetailedDescription: sizeDetailedDescription,
	name: "size",
	args: "[+|-]n[ckMGTP]",
	parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
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
	},
})

const sizeDetailedDescription = `
-size [+|-]n[ckMGTP]

Returns true if the entry's size attribute is n 512-byte blocks,
rounded up to the nearest block. If n is suffixed with a unit,
then the raw size is compared to n scaled as:

c        character (byte)
k        kibibytes (1024 bytes)
M        mebibytes (1024 kibibytes)
G        gibibytes (1024 mebibytes)
T        tebibytes (1024 gibibytes)
P        pebibytes (1024 tebibytes)

If n is prefixed with a +/-, then the comparison returns true if
the size is greater-than/less-than n.

Examples:
  -size 2        Returns true if the entry's size is 2 512-byte blocks,
                 rounded up to the nearest block

  -size +2       Returns true if the entry's size is greater than 2
                 512-byte blocks, rounded up to the nearest block

  -size -2       Returns true if the entry's size is less than 2
                 512-byte blocks, rounded up to the nearest block

  -size +1k      Returns true if the entry's size is greater than 1
                 kibibyte
`
