package predicate

import (
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/shopspring/decimal"
)

// NPE_StringPredicate returns a node representing NPE StringPredicate
func NPE_StringPredicate() rql.StringPredicate {
	return expression.New("StringPredicate", true, func() rql.ASTNode {
		return String()
	}).(rql.StringPredicate)
}

// NPE_TimePredicate returns a node representing NPE TimePredicate
func NPE_TimePredicate() rql.TimePredicate {
	return expression.New("TimePredicate", true, func() rql.ASTNode {
		return Time("", time.Time{})
	}).(rql.TimePredicate)
}

// NPE_UnsignedNumericPredicate returns a node representing NPE UnsignedNumericPredicate
func NPE_UnsignedNumericPredicate() rql.NumericPredicate {
	return expression.New("UnsignedNumericPredicate", true, func() rql.ASTNode {
		return UnsignedNumeric("", decimal.Decimal{})
	}).(rql.NumericPredicate)
}

// NPE_NumericPredicate returns a node representing NPE NumericPredicate
func NPE_NumericPredicate() rql.NumericPredicate {
	return expression.New("NumericPredicate", true, func() rql.ASTNode {
		return Numeric("", decimal.Decimal{})
	}).(rql.NumericPredicate)
}

// NPE_ValuePredicate returns a node representing NPE ValuePredicate
func NPE_ValuePredicate() rql.ValuePredicate {
	return expression.New("ValuePredicate", true, func() rql.ASTNode {
		return internal.NewNonterminalNode(
			Object(),
			Array(),
			Null(),
			Boolean(false),
			NumericValue(NPE_NumericPredicate()),
			TimeValue(NPE_TimePredicate()),
			StringValue(NPE_StringPredicate()),
		)
	}).(rql.ValuePredicate)
}
