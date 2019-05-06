package numeric

import (
	"fmt"
	"strconv"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
)

// ParsePositiveInt parses a positive integer.
func ParsePositiveInt(str string) (int64, error) {
	// Use ParseInt instead of ParseUint because uint => int
	// conversion leads to large uint values being interpreted
	// as large negative int values.
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return n, errz.NewMatchError(err.Error())
	}
	if n < 0 {
		return 0, fmt.Errorf("expected a positive number, got %v", n)
	}
	return n, err
}

// Negate returns a new parser g that negates any number parsed
// by p. For example, if p parses "15" as "15", then g parses
// "15" as "-15".
func Negate(p Parser) Parser {
	return func(str string) (int64, error) {
		n, err := p(str)
		if err != nil {
			return 0, err
		}
		return -n, nil
	}
}
