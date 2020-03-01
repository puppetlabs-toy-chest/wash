package ast

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/primary"
)

// Primary returns an AST node representing PE Primary
func Primary() rql.ASTNode {
	nt := internal.NewNonterminalNode(
		primary.Action(PE_ActionPredicate()),
		primary.Name(PE_StringPredicate()),
		primary.CName(PE_StringPredicate()),
		primary.Path(PE_StringPredicate()),
		primary.Kind(PE_StringPredicate()),
		primary.Atime(PE_TimePredicate()),
		primary.Crtime(PE_TimePredicate()),
		primary.Ctime(PE_TimePredicate()),
		primary.Mtime(PE_TimePredicate()),
		primary.Size(PE_UnsignedNumericPredicate()),
	)
	nt.SetMatchErrMsg("expected a primary")
	return nt
}
