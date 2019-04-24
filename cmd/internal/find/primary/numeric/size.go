package numeric

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

var bytesMap = map[byte]int64{
	'c': 1,
	'k': 1024,
	'M': 1024 * 1024,
	'G': 1024 * 1024 * 1024,
	'T': 1024 * 1024 * 1024 * 1024,
	'P': 1024 * 1024 * 1024 * 1024 * 1024,
}

// BytesOf returns the number of bytes specified by the given size
// unit. Valid size units are:
//   c => character (byte)
//   k => kibibyte
//   M => mebibyte
//   G => gibibyte
//   T => tebibyte
//   P => pebibyte
//
func BytesOf(unit byte) int64 {
	if b, ok := bytesMap[unit]; ok {
		return b
	}
	panic(fmt.Sprintf("numeric.BytesOf received an unexpected unit %v", unit))
}

// SizeRegex describes a valid size value. See BytesOf for more details on
// what the individual units represent.
var SizeRegex = regexp.MustCompile(`^\d+[ckMGTP]$`)

// ParseSize parses a size value. Size values are described by SizeRegex.
func ParseSize(str string) (int64, error) {
	if !SizeRegex.MatchString(str) {
		msg := fmt.Sprintf("size values must conform to the regex `%v`", SizeRegex)
		return 0, errz.NewMatchError(msg)
	}
	endIx := len(str) - 1
	n, err := strconv.ParseInt(str[0:endIx], 10, 64)
	if err != nil {
		// We should never hit this code-path because SizeRegex already validated n.
		return 0, err
	}
	unit := str[endIx]
	return n * BytesOf(unit), nil
}
