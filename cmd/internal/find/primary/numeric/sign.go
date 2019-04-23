package numeric

import (
	"fmt"
	"strconv"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
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

// Negate returns a new parser g that parses all numbers
// satifying the regex `{<number>}` where <number> is s.t.
// p(<number>) does not return an error. The returned number
// is the negation of the parsed number. For example, if p
// parses "15" as the number 15, then g parses "{15}" as the
// number "-15".
//
// Note that g returns a syntax error if p(<number>) returns
// a match error.
func Negate(p Parser) Parser {
	return func(str string) (int64, error) {
		if len(str) == 0 {
			return 0, errz.NewMatchError("expected a number")
		}
		if str[0] != '{' {
			msg := "expected an opening '{'"
			if str[0] == '}' {
				return 0, fmt.Errorf(msg)
			}
			return 0, errz.NewMatchError(msg)
		}
		str = str[1:]
		endIx := len(str) - 1
		if endIx < 0 {
			return 0, fmt.Errorf("expected a closing '}'")
		}
		if str[endIx] != '}' {
			return 0, fmt.Errorf("expected a closing '}'")
		}
		str = str[0:endIx]
		n, err := p(str)
		if err != nil {
			if errz.IsMatchError(err) {
				return 0, fmt.Errorf("expected a number inside '{}', got: %v", str)
			}
			return 0, err
		}
		return -n, nil
	}
}
