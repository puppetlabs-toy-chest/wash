package numeric

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/errz"
)

// Bracket returns a new parser g that parses all numbers
// satifying the regex `{<number>}` where <number> is s.t.
// p(<number>) does not return an error. For example, if p
// parses "15" as the number 15, then g parses "{15}" as the
// number "15". Bracket's typically used to implement negation
// semantics. For example, Bracket(Negate(ParsePositiveInt))
// returns a parser that parses all n <= 0, where each n is
// represented as {m} where m = -n.
//
// Note that g returns a syntax error if p(<number>) returns
// a match error.
func Bracket(p Parser) Parser {
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
		return n, nil
	}
}