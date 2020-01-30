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
	return expression.New("Primary", func() rql.ASTNode {
		return Primary()
	}).(rql.Primary)
}

// PE_ActionPredicate returns a node representing PE ActionPredicate
func PE_ActionPredicate() rql.ActionPredicate {
	return expression.New("ActionPredicate", func() rql.ASTNode {
		return predicate.Action(plugin.Action{})
	}).(rql.ActionPredicate)
}

// PE_StringPredicate returns a node representing PE StringPredicate
func PE_StringPredicate() rql.StringPredicate {
	return expression.New("StringPredicate", func() rql.ASTNode {
		return predicate.String()
	}).(rql.StringPredicate)
}

// PE_TimePredicate returns a node representing PE TimePredicate
func PE_TimePredicate() rql.TimePredicate {
	return expression.New("TimePredicate", func() rql.ASTNode {
		return predicate.Time("", time.Time{})
	}).(rql.TimePredicate)
}

// PE_UnsignedNumericPredicate returns a node representing PE UnsignedNumericPredicate
func PE_UnsignedNumericPredicate() rql.NumericPredicate {
	return expression.New("UnsignedNumericPredicate", func() rql.ASTNode {
		return predicate.UnsignedNumeric("", decimal.Decimal{})
	}).(rql.NumericPredicate)
}
