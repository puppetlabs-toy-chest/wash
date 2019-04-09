package munge

import (
	"fmt"
)

func intToSize(size int64) (uint64, error) {
	if size < 0 {
		return 0, fmt.Errorf("%v is a negative size", size)
	}
	return uint64(size), nil
}

// ToSize converts v to a uint64 that's meant to represent
// content size
func ToSize(v interface{}) (uint64, error) {
	switch sz := v.(type) {
	case uint64:
		return sz, nil
	case int:
		return intToSize(int64(sz))
	case int32:
		return intToSize(int64(sz))
	case int64:
		return intToSize(sz)
	case float64:
		if sz != float64(int64(sz)) {
			return 0, fmt.Errorf("%v is a decimal size", sz)
		}
		return intToSize(int64(sz))
	default:
		return 0, fmt.Errorf("%v is not a valid size type. Valid size types are uint64, int, int32, int64, float64", v)
	}
}
