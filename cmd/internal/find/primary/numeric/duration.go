package numeric

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

var durationsMap = map[byte]time.Duration{
	's': time.Second,
	'm': time.Minute,
	'h': time.Hour,
	'd': 24 * time.Hour,
	'w': 7 * 24 * time.Hour,
}

// DurationOf returns the number of seconds specified by the given duration
// unit. Valid duration units are:
//   s => second
//   m => minute
//   h => hour
//   d => day
//   w => week
//
func DurationOf(unit byte) int64 {
	if d, ok := durationsMap[unit]; ok {
		return int64(d)
	}
	panic(fmt.Sprintf("util.DurationOf received an unexpected unit %v", unit))
}

var durationChunkRegex = regexp.MustCompile(`\d+[smhdw]`)

// DurationRegex describes a valid duration value. See DurationOf for more
// details on what the individual units represent.
var DurationRegex = regexp.MustCompile(
	"^(" + durationChunkRegex.String() + ")+$",
)

// ParseDuration parses a duration value as an int64. Duration values are
// described by DurationRegex.
func ParseDuration(str string) (int64, error) {
	if !DurationRegex.MatchString(str) {
		msg := fmt.Sprintf("duration values must conform to the regex `%v`", DurationRegex)
		return 0, errz.NewMatchError(msg)
	}
	var duration int64
	for _, chunk := range durationChunkRegex.FindAllString(str, -1) {
		endIx := len(chunk) - 1
		unit := chunk[endIx]
		n, err := strconv.ParseInt(chunk[0:endIx], 10, 64)
		if err != nil {
			// We should never hit this code-path because DurationRegex already validated n.
			return 0, err
		}
		duration += n * DurationOf(unit)
	}
	return duration, nil
}
