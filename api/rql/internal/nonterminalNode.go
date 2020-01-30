package internal

import "github.com/puppetlabs/wash/api/rql"

import "github.com/puppetlabs/wash/api/rql/internal/errz"

// A NonterminalNode is a node that can be matched by one or
// more other nodes when it is unmarshaled. MatchedNode returns
// the matched node
type NonterminalNode interface {
	rql.ASTNode
	MatchedNode() rql.ASTNode
	SetMatchedNode(rql.ASTNode) NonterminalNode
	SetMatchErrMsg(msg string) NonterminalNode
}

func NewNonterminalNode(n rql.ASTNode, ns ...rql.ASTNode) NonterminalNode {
	return &nonterminalNode{
		nodes: append(ns, n),
	}
}

type nonterminalNode struct {
	matchedNode rql.ASTNode
	errMsg      string
	nodes       []rql.ASTNode
}

func (nt *nonterminalNode) Marshal() interface{} {
	return nt.matchedNode.Marshal()
}

func (nt *nonterminalNode) Unmarshal(input interface{}) error {
	for _, n := range nt.nodes {
		err := n.Unmarshal(input)
		if err == nil {
			nt.SetMatchedNode(n)
			return nil
		}
		if !errz.IsMatchError(err) {
			return err
		}
	}
	return errz.MatchErrorf(nt.errMsg)
}

func (nt *nonterminalNode) MatchedNode() rql.ASTNode {
	return nt.matchedNode
}

func (nt *nonterminalNode) SetMatchedNode(n rql.ASTNode) NonterminalNode {
	if mnt, ok := n.(NonterminalNode); ok {
		n = mnt.MatchedNode()
	}
	nt.matchedNode = n
	return nt
}

func (nt *nonterminalNode) SetMatchErrMsg(errMsg string) NonterminalNode {
	nt.errMsg = errMsg
	return nt
}

var _ = NonterminalNode(&nonterminalNode{})
