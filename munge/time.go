package munge

import (
	"fmt"
	"time"

	"github.com/araddon/dateparse"
)

func unixSecondsToTime(s int64) time.Time {
	if s <= 0 {
		return time.Time{}
	}
	return time.Unix(s, 0)
}

// ToTime converts v to a time.Time object.
func ToTime(v interface{}) (time.Time, error) {
	switch t := v.(type) {
	case time.Time:
		return t, nil
	case int64:
		return unixSecondsToTime(t), nil
	case float64:
		if t != float64(int64(t)) {
			return time.Time{}, fmt.Errorf("could not convert to time.Time: the provided time %v is a decimal number", t)
		}
		return unixSecondsToTime(int64(t)), nil
	case string:
		// time.Time objects are marshaled into RFC3339 formatted strings.
		// Thus, try parsing in that format first since it is the common
		// case. If that fails, then delegate to the dateparse library.
		// We do this for performance reasons.
		tm, err := time.Parse(time.RFC3339, t)
		if err != nil {
			tm, err = dateparse.ParseAny(t)
			if err != nil {
				err = fmt.Errorf("could not parse %v into a time.Time object: %v", t, err)
			}
		}
		return tm, err
	default:
		return time.Time{}, fmt.Errorf("%v is not a valid time.Time type. Valid time.Time types are time.Time, int64, and string", v)
	}
}
