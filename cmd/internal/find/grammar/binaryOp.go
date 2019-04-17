package grammar

import "github.com/puppetlabs/wash/cmd/internal/find/types"

// BinaryOp represents a binary operator in a `wash find` expression
type BinaryOp struct {
	Tokens     []string
	Precedence int
	Combine    func(p1 types.Predicate, p2 types.Predicate) types.Predicate
}

// BinaryOps is a map of <token> => <binaryOp>. This is populated by NewBinaryOp.
var BinaryOps = make(map[string]*BinaryOp)

// NewBinaryOp creates a new binary op. When creating a new binary op with this function
// via a package variable assignment, be sure to comment nolint above the
// variable so that CI does not mark it as unused. See andOp in find/operators.go
// for an example.
func NewBinaryOp(tokens []string, precedence int, combine func(p1 types.Predicate, p2 types.Predicate) types.Predicate) *BinaryOp {
	b := &BinaryOp{
		Tokens:     tokens,
		Precedence: precedence,
		Combine:    combine,
	}
	for _, t := range tokens {
		BinaryOps[t] = b
	}
	return b
}
