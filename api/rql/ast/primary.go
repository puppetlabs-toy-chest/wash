package ast

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/primary"
)

// Primary returns an AST node representing PE Primary
func Primary() rql.ASTNode {
	nt := internal.NewNonterminalNode(
		primary.Action(NPE_ActionPredicate()),
		primary.Name(NPE_StringPredicate()),
		primary.CName(NPE_StringPredicate()),
		primary.Path(NPE_StringPredicate()),
		primary.Kind(NPE_StringPredicate()),
		primary.Atime(NPE_TimePredicate()),
		primary.Crtime(NPE_TimePredicate()),
		primary.Ctime(NPE_TimePredicate()),
		primary.Mtime(NPE_TimePredicate()),
		primary.Size(NPE_UnsignedNumericPredicate()),
	)
	nt.SetMatchErrMsg("expected a primary")
	return nt
}
