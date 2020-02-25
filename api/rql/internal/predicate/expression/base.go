package expression

import "github.com/puppetlabs/wash/api/rql"

import "github.com/puppetlabs/wash/api/rql/internal"

type expressionNode interface {
	rql.ASTNode
	valid() bool
}

// base is a base class for expression nodes
type base struct{}

func (b *base) IsPrimary() bool {
	return true
}

func (b *base) valid() bool {
	return true
}

// unravelNTN => unravelNonterminalNode
func unravelNTN(p rql.ASTNode) rql.ASTNode {
	if nt, ok := p.(internal.NonterminalNode); ok {
		return nt.MatchedNode()
	}
	return p
}
