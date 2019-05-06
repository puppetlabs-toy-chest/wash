package primary

import (
	"fmt"
	"math"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
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
//
// Example inputs:
//   -mtime 1      (true if the difference between the entry's mtime and startTime is exactly 1 24-hour period)
//   -mtime +1     (true if the difference between the entry's mtime and startTime is greater than 1 24-hour period)
//   -mtime -1     (true if the difference between the entry's mtime and startTime is less than 1 24-hour period)
//   -mtime +1h30m (true if the difference between the entry's mtime and startTime is greater than 1 hour and 30 minutes)
//
// NOTE: For non-unit time values (e.g. 1, +1, -1), the difference is rounded to the next 24-hour period. For
// example, a difference of 1.5 days will be rounded to 2 days.
func newTimeAttrPrimary(name string) *primary {
	tk := "-" + name
	return Parser.newPrimary([]string{tk}, func(tokens []string) (predicate.Entry, []string, error) {
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
	})
}

//nolint
var ctimePrimary = newTimeAttrPrimary("ctime")

//nolint
var mtimePrimary = newTimeAttrPrimary("mtime")

//nolint
var atimePrimary = newTimeAttrPrimary("atime")
