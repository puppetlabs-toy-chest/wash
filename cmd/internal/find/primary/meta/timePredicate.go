package meta

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/munge"
)

// TimePredicate => (+|-)? Duration
// Duration      => numeric.DurationRegex | '{' numeric.DurationRegex '}'
func parseTimePredicate(tokens []string) (predicate, []string, error) {
	if params.StartTime.IsZero() {
		panic("meta.parseTimePredicate called without setting params.StartTime")
	}
	if len(tokens) == 0 {
		return nil, nil, errz.NewMatchError("expected a +, -, or a digit")
	}
	token := tokens[0]
	numericP, parserID, err := numeric.ParsePredicate(
		token,
		numeric.ParseDuration,
		numeric.Negate(numeric.ParseDuration),
	)
	if err != nil {
		if errz.IsMatchError(err) {
			msg := fmt.Sprintf("expected a duration but got %v", token)
			return nil, nil, errz.NewMatchError(msg)
		}
		// err is a parse error, so return it.
		return nil, nil, err
	}
	if parserID == 1 {
		// User passed-in something like +{1h}. This means they want to
		// base the predicate off of 'timeV - StartTime' instead of
		// 'StartTime - timeV'. For our example of +{1h}, we want numericP
		// to return true if 'timeV - StartTime' > 1h. Unfortunately, there
		// isn't a clean way to mathematically enforce this without complicating
		// the parsing logic. However, we can get a pretty good approximation
		// by inverting numericP. For the +{1h} example, inverting numericP means
		// that numericP returns true if 'StartTime - timeV' <= -1h which reduces
		// to returning true if 'timeV - StartTime' >= 1h. This is not quite correct
		// because numericP will return true if timeV == startTime, but given that
		// Go's time.Time objects have nanosecond precision, this is a very rare
		// (if not impossible) edge case so negating numericP should be good enough.
		numericP = numeric.Not(numericP)
	}
	p := func(v interface{}) bool {
		timeV, err := munge.ToTime(v)
		if err != nil {
			return false
		}
		diff := int64(params.StartTime.Sub(timeV))
		return numericP(diff)
	}
	return p, tokens[1:], nil
}
