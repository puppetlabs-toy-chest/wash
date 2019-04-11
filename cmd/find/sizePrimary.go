package cmdfind

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	apitypes "github.com/puppetlabs/wash/api/types"
)

// Use bytesOf instead of a hash so that we generate a more readable
// panic message
func bytesOf(unit byte) uint64 {
	switch unit {
	case 'c':
		return 1
	case 'k':
		return 1024
	case 'M':
		return 1024 * 1024
	case 'G':
		return 1024 * 1024 * 1024
	case 'T':
		return 1024 * 1024 * 1024 * 1024
	case 'P':
		return 1024 * 1024 * 1024 * 1024 * 1024
	default:
		panic(fmt.Sprintf("cmdfind.bytesOf received an unexpected unit %v", unit))
	}
}

var sizeValueRegex = regexp.MustCompile(`^(\+|-)?((\d+)|(\d+[ckMGTP]))$`)

// sizePrimary => -size (+|-)?((\d+)|(\d+[ckMGTP]))
//
// where c => character (byte), k => kilobyte, M => megabyte, G => gigabyte, T => terabyte, P => petabyte
//
// Example inputs:
//   -size 2   (true if the entry's size in 512-byte blocks, rounded up, is 2)
//   -size +2  (true if the entry's size in 512-byte blocks, rounded up, is greater than 2)
//   -size -2  (true if the entry's size in 512-byte blocks, rounded up, is less than 2)
//   -size +1k (true if the entry's size is greater than 1 kilobyte (1024 bytes))
//
//nolint
var sizePrimary = newAtom([]string{"-size"}, func(tokens []string) (Predicate, []string, error) {
	tokens = tokens[1:]
	if len(tokens) == 0 {
		return nil, nil, fmt.Errorf("-size: requires additional arguments")
	}
	v := tokens[0]
	if !sizeValueRegex.MatchString(v) {
		return nil, nil, fmt.Errorf("-size: %v: illegal size value", v)
	}

	cmp := v[0]
	if cmp == '+' || cmp == '-' {
		v = strings.TrimPrefix(v, string(cmp))
	} else {
		cmp = '='
	}

	p := func(e *apitypes.ListEntry) bool {
		if !e.Attributes.HasSize() {
			return false
		}
		size := e.Attributes.Size()

		var parsedSize uint64
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			// v (n) is an integer, so convert the size to the # of 512-byte blocks (rounded up).
			size = uint64(math.Ceil(float64(size) / 512.0))
			parsedSize = n
		} else {
			// v is followed by a scale indicator
			endIx := len(v) - 1
			unit := v[endIx]
			n, err := strconv.ParseUint(v[0:endIx], 10, 32)
			if err != nil {
				// We should never hit this code-path because sizeValueRegex
				// already verified that v[0:endIx] is an integer.
				msg := fmt.Sprintf("errored parsing size %v, which is an expected integer: %v", v[0:endIx], err)
				panic(msg)
			}
			parsedSize = n * bytesOf(unit)
		}

		switch cmp {
		case '+':
			return size > parsedSize
		case '-':
			return size < parsedSize
		default:
			return size == parsedSize
		}
	}
	return p, tokens[1:], nil
})
