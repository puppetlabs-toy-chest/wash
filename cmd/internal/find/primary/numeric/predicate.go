package numeric

import (
	"fmt"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/errz"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
)

// Predicate represents a Numeric predicate
type Predicate func(int64) bool

// And returns p1 && p2
func (p1 Predicate) And(p2 predicate.Predicate) predicate.Predicate {
	return Predicate(func(n int64) bool {
		return p1(n) && (p2.(Predicate))(n)
	})
}

// Or returns p1 || p2
func (p1 Predicate) Or(p2 predicate.Predicate) predicate.Predicate {
	return Predicate(func(n int64) bool {
		return p1(n) || (p2.(Predicate))(n)
	})
}

// Negate returns Not(p1)
func (p1 Predicate) Negate() predicate.Predicate {
	return Predicate(func(n int64) bool {
		return !p1(n)
	})
}

// IsSatisfiedBy returns true if v satisfies the predicate, false otherwise
func (p1 Predicate) IsSatisfiedBy(v interface{}) bool {
	n, ok := v.(int64)
	if !ok {
		return false
	}
	return p1(n)
}

// Parser parses numeric values.
type Parser func(string) (int64, error)

// ParsePredicate parses a numeric predicate from str. Str should
// satisfy the regex `(\+|\-)?<number>`, where <number> is s.t.
// that parser(<number>) does not return an error for at least one
// parser in parsers. The returned value is the parsed predicate
// and the id of the parser that parsed <number>.
func ParsePredicate(str string, parsers ...Parser) (Predicate, int, error) {
	if len(str) == 0 {
		return nil, -1, errz.NewMatchError("empty input")
	}
	if len(parsers) == 0 {
		panic("numeric.ParsePredicate called without any parsers")
	}

	// TODO: Introduce "+="/"-=" to represent ">="/"<="?
	cmp := str[0]
	if cmp == '+' || cmp == '-' {
		str = str[1:]
	} else {
		cmp = '='
	}

	var parserID int
	var n int64
	var err error
	for i, parser := range parsers {
		n, err = parser(str)
		if err == nil {
			parserID = i
			break
		}
		if !errz.IsMatchError(err) {
			// Parser matched the input, but returned a parse error. Return
			// the error.
			return nil, -1, err
		}
	}
	if err != nil {
		msg := fmt.Sprintf("%v is not a number", str)
		return nil, -1, errz.NewMatchError(msg)
	}

	return func(v int64) bool {
		switch cmp {
		case '+':
			return v > n
		case '-':
			return v < n
		default:
			return v == n
		}
	}, parserID, nil
}
