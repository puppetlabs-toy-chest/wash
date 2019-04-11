package cmdfind

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"time"

	apitypes "github.com/puppetlabs/wash/api/types"
)

var startTime time.Time

// SetStartTime sets `wash find`'s start time. This is primarily
// needed by the ctime, mtime, and atime primaries, which base their
// predicates off the set value.
func SetStartTime(time time.Time) {
	startTime = time
}

var durationsMap = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

// Use durationOf instead of a hash so that we generate a more readable
// panic message
func durationOf(unit byte) time.Duration {
	if d, ok := durationsMap[unit]; ok {
		return d
	}
	panic(fmt.Sprintf("cmdfind.durationOf received an unexpected unit %v", unit))
}

// We use getTimeAttrValue to retrieve the time attribute's value for performance
// reasons. Using e.Attributes.ToMap()[name] would be slower because it would
// require an additional type assertion to extract the time.Time object.
func getTimeAttrValue(name string, e *apitypes.ListEntry) (time.Time, bool) {
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

var timeAttrValueRegex = regexp.MustCompile(`^(\+|-)?((\d+)|(\d+[smhdw])+)$`)

// Describes values like 1h30m, 3w2d, etc.
var timeAttrUnitValueRegex = regexp.MustCompile(`\d+[smhdw]`)

// parseDuration is a helper for newTimeAttrPrimary that parses the
// duration from v. It returns the parsed duration, and a boolean
// indicating whether the difference between the entry's time attribute
// and start time needs to be rounded to the next 24-hour period (which
// is only true if v is an integer).
func parseDuration(v string) (time.Duration, bool) {
	var roundDiff bool
	var duration time.Duration
	if n, err := strconv.ParseInt(v, 10, 32); err == nil {
		// v (n) is an integer
		duration = time.Duration(n) * durationOf('d')
		roundDiff = true
	} else {
		// v consists of individual time units. Add them up to get the overall
		// duration. Note that repeated units will be stacked, meaning something
		// like "1h1h1h1h" will result in a duration of 4 hours. This behavior matches
		// BSD's find command's behavior.
		for _, chunk := range timeAttrUnitValueRegex.FindAllString(v, -1) {
			endIx := len(chunk) - 1
			unit := chunk[endIx]
			n, err := strconv.ParseInt(chunk[0:endIx], 10, 32)
			if err != nil {
				// We should never hit this code-path because timeAttrUnitValueRegex
				// already verified that chunk[0:endIx] is an integer.
				msg := fmt.Sprintf("errored parsing duration %v, which is an expected integer: %v", chunk[0:endIx], err)
				panic(msg)
			}
			duration += time.Duration(n) * durationOf(unit)
		}
	}
	return duration, roundDiff
}

// timeAttrPrimary => -<name> (+|-)?((\d+)|(\d+[smhdw])
//
// where s => seconds, m => minutes, h => hours, d => days, w => weeks
//
// Example inputs:
//   -mtime 1      (true if the difference between the entry's mtime and startTime is exactly 1 24-hour period)
//   -mtime +1     (true if the difference between the entry's mtime and startTime is greater than 1 24-hour period)
//   -mtime -1     (true if the difference between the entry's mtime and startTime is less than 1 24-hour period)
//   -mtime +1h30m (true if the difference between the entry's mtime and startTime is greater than 1 hour and 30 minutes)
//
// NOTE: For non-unit time values (e.g. 1, +1, -1), the difference is rounded to the next 24-hour period. For
// example, a difference of 1.5 days will be rounded to 2 days.
func newTimeAttrPrimary(name string) *atom {
	tk := "-" + name
	return newAtom([]string{tk}, func(tokens []string) (Predicate, []string, error) {
		if startTime == (time.Time{}) {
			panic("Attempting to parse a time primary without calling cmdfind.SetStartTime")
		}

		tokens = tokens[1:]
		if len(tokens) == 0 {
			return nil, nil, fmt.Errorf("%v: requires additional arguments", tk)
		}
		v := tokens[0]
		if !timeAttrValueRegex.MatchString(v) {
			return nil, nil, fmt.Errorf("%v: %v: illegal time value", tk, v)
		}

		cmp := v[0]
		if cmp == '+' || cmp == '-' {
			v = v[1:]
		} else {
			cmp = '='
		}

		duration, roundDiff := parseDuration(v)
		p := func(e *apitypes.ListEntry) bool {
			t, ok := getTimeAttrValue(name, e)
			if !ok {
				return false
			}
			diff := startTime.Sub(t)
			if roundDiff {
				// Round-up diff to the next 24-hour period
				roundedDays := time.Duration(math.Ceil(float64(diff) / float64(durationOf('d'))))
				diff = roundedDays * durationOf('d')
			}
			switch cmp {
			case '+':
				return diff > duration
			case '-':
				return diff < duration
			default:
				return diff == duration
			}
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
