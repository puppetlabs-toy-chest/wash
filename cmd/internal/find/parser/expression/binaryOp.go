package expression

import "github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"

// BinaryOp represents a binary operator in a predicate expression
type BinaryOp struct {
	tokens     []string
	precedence int
	combine    func(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate
}

func newBinaryOp(tokens []string, precedence int, combine func(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate) *BinaryOp {
	return &BinaryOp{
		tokens:     tokens,
		precedence: precedence,
		combine:    combine,
	}
}

var andOp = newBinaryOp([]string{"-a", "-and"}, 1, func(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
		return p1.And(p2)
})

var orOp = newBinaryOp([]string{"-o", "-or"}, 0, func(p1 predicate.Predicate, p2 predicate.Predicate) predicate.Predicate {
		return p1.Or(p2)
})
