package expression

import "github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"

// BinaryOp represents a binary operator in a predicate expression
type BinaryOp struct {
	tokens     []string
	precedence int
	op         predicate.BinaryOp
}

func newBinaryOp(tokens []string, precedence int, op predicate.BinaryOp) *BinaryOp {
	return &BinaryOp{
		tokens:     tokens,
		precedence: precedence,
		op:         op,
	}
}

func newAndOp(op predicate.BinaryOp) *BinaryOp {
	return newBinaryOp([]string{"-a", "-and"}, 1, op)
}

func newOrOp(op predicate.BinaryOp) *BinaryOp {
	return newBinaryOp([]string{"-o", "-or"}, 0, op)
}
