package ast

import (
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

// PE_Primary returns a node representing a predicate expression (PE)
// of primaries
func PE_Primary() rql.Primary {
	return expression.New("Primary", false, func() rql.ASTNode {
		return Primary()
	}).(rql.Primary)
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
	return expression.New("StringPredicate", true, func() rql.ASTNode {
		return predicate.String()
	}).(rql.StringPredicate)
}

// NPE_TimePredicate returns a node representing NPE TimePredicate
func NPE_TimePredicate() rql.TimePredicate {
	return expression.New("TimePredicate", true, func() rql.ASTNode {
		return predicate.Time("", time.Time{})
	}).(rql.TimePredicate)
}

// NPE_UnsignedNumericPredicate returns a node representing NPE UnsignedNumericPredicate
func NPE_UnsignedNumericPredicate() rql.NumericPredicate {
	return expression.New("UnsignedNumericPredicate", true, func() rql.ASTNode {
		return predicate.UnsignedNumeric("", decimal.Decimal{})
	}).(rql.NumericPredicate)
}
