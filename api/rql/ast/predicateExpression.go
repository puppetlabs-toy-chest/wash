package ast

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/puppetlabs/wash/plugin"
)

// PE_Primary returns a node representing a predicate expression (PE)
// of primaries
func PE_Primary() rql.Primary {
	return expression.New("Primary", false, func() rql.ASTNode {
		return Primary()
	}).(rql.Primary)
}

// PE_Object returns a node representing PE Object
func PE_Object() rql.ValuePredicate {
	return expression.New("ObjectPredicate", false, func() rql.ASTNode {
		return predicate.Object()
	}).(rql.ValuePredicate)
}

// NPE_ActionPredicate returns a node representing a negatable predicate
// expression (NPE) of ActionPredicate
func NPE_ActionPredicate() rql.ActionPredicate {
	return expression.New("ActionPredicate", true, func() rql.ASTNode {
		return predicate.Action(plugin.Action{})
	}).(rql.ActionPredicate)
}

// NPE_StringPredicate returns a node representing NPE StringPredicate
func NPE_StringPredicate() rql.StringPredicate {
	return predicate.NPE_StringPredicate()
}

// NPE_TimePredicate returns a node representing NPE TimePredicate
func NPE_TimePredicate() rql.TimePredicate {
	return predicate.NPE_TimePredicate()
}

// NPE_UnsignedNumericPredicate returns a node representing NPE UnsignedNumericPredicate
func NPE_UnsignedNumericPredicate() rql.NumericPredicate {
	return predicate.NPE_UnsignedNumericPredicate()
}
