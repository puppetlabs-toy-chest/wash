package primary

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// We use getTimeAttrValue to retrieve the time attribute's value for performance
// reasons. Using e.Attributes.ToMap()[name] would be slower because it would
// require an additional type assertion to extract the time.Time object.
func getTimeAttrValue(name string, e types.Entry) (time.Time, bool) {
	switch name {
	case "ctime":
		return e.Attributes.Ctime(), e.Attributes.HasCtime()
	case "mtime":
		return e.Attributes.Mtime(), e.Attributes.HasMtime()
	case "atime":
		return e.Attributes.Atime(), e.Attributes.HasAtime()
	default:
		panic(fmt.Sprintf("cmdfind.getTimeAttrValue called with nonexistent time attribute %v", name))
	}
}

// timeAttrPrimary => -<name> (+|-)?(\d+ | (numeric.DurationRegex)+)
func newTimeAttrPrimary(name string) *Primary {
	return Parser.add(&Primary{
		Description: fmt.Sprintf("Returns true if the entry's %v attribute satisfies the given time predicate", name),
		DetailedDescription: timeAttrDetailedDescription(name),
		name: name,
		args: "[+|-]n[smhdw]",
		parseFunc: func(tokens []string) (types.EntryPredicate, []string, error) {
			if params.StartTime.IsZero() {
				panic("Attempting to parse a time primary without setting params.StartTime")
			}
			if len(tokens) == 0 {
				return nil, nil, fmt.Errorf("requires additional arguments")
			}
			numericP, parserID, err := numeric.ParsePredicate(
				tokens[0],
				numeric.ParsePositiveInt,
				numeric.ParseDuration,
			)
			if err != nil {
				return nil, nil, fmt.Errorf("%v: illegal time value", tokens[0])
			}

			p := func(e types.Entry) bool {
				t, ok := getTimeAttrValue(name, e)
				if !ok {
					return false
				}
				diff := int64(params.StartTime.Sub(t))
				if parserID == 0 {
					// n was an integer, so round-up diff to the next 24-hour period
					diff = int64(math.Ceil(float64(diff) / float64(numeric.DurationOf('d'))))
				}
				return numericP(diff)
			}
			return p, tokens[1:], nil
		},
	})
}

func timeAttrDetailedDescription(name string) string {
	// Note that some of the spacing is purposefully mis-aligned
	// because {name} is replaced with the name parameter, which
	// is not the same # of characters as the string literal
	// '{name}'
	descr := `
-{name} [+|-]n[smhdw]

Returns true if the entry's {name} attribute is exactly n days,
rounded up to the nearest day. If n is suffixed with a unit, then
the raw {name} is compared to n scaled as:

s        second
m        minute (60 seconds)
h        hour   (60 minutes)
d        day    (24 hours)
w        week   (7 days)

If n is prefixed with a +/-, then the comparison returns true if the
{name} is greater-than/less-than n.

Examples:
  -{name} 1        Returns true if the entry's {name} is exactly 1
                  day, rounded up to the nearest day

  -{name} +1       Returns true if the entry's {name} is more than 1
                  day ago, rounded up to the nearest day

  -{name} -1       Returns true if the entry's {name} is less than 1
                  day ago, rounded up to the nearest day

  -{name} +1h      Returns true if the entry's {name} is more than one
                  hour ago

NOTE: All comparisons are made with respect to the find command's
start time.
`
	return strings.NewReplacer("{name}", name).Replace(descr)
}

//nolint
var ctimePrimary = newTimeAttrPrimary("ctime")

//nolint
var mtimePrimary = newTimeAttrPrimary("mtime")

//nolint
var atimePrimary = newTimeAttrPrimary("atime")
