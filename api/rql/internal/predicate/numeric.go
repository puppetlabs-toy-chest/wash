package predicate

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
	"github.com/shopspring/decimal"
)

func Numeric(op ComparisonOp, n decimal.Decimal) rql.NumericPredicate {
	return &numeric{
		op: op,
		n:  n,
	}
}

func UnsignedNumeric(op ComparisonOp, n decimal.Decimal) rql.NumericPredicate {
	p := Numeric(op, n).(*numeric)
	p.unsigned = true
	return p
}

type numeric struct {
	op       ComparisonOp
	n        decimal.Decimal
	unsigned bool
}

func (p *numeric) Marshal() interface{} {
	return []interface{}{string(p.op), p.n.String()}
}

func (p *numeric) Unmarshal(input interface{}) error {
	m := matcher.Array(func(v interface{}) bool {
		opStr, ok := v.(string)
		return ok && comparisonOpMap[ComparisonOp(opStr)]
	})
	if !m(input) {
		return errz.MatchErrorf("must be formatted as [<comparison_op>, <number>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [<comparison_op>, <number>]")
	}
	if len(array) != 2 {
		return fmt.Errorf("must be formatted as [<comparison_op>, <number>] (missing the number)")
	}
	op := ComparisonOp(array[0].(string))
	var n decimal.Decimal
	var err error
	switch t := array[1].(type) {
	case float64:
		n = decimal.NewFromFloat(t)
	case string:
		n, err = decimal.NewFromString(t)
		if err != nil {
			return fmt.Errorf("failed to parse %v as a number: %w", t, err)
		}
	default:
		return fmt.Errorf("%v is not a valid number", t)
	}
	p.op = op
	if p.unsigned && n.LessThan(decimal.Zero) {
		return fmt.Errorf("%v must be an unsigned (non-negative) number", n)
	}
	p.n = n
	return nil
}

func (p *numeric) EvalNumeric(n decimal.Decimal) bool {
	switch p.op {
	case LT:
		return n.LessThan(p.n)
	case LTE:
		return n.LessThanOrEqual(p.n)
	case GT:
		return n.GreaterThan(p.n)
	case GTE:
		return n.GreaterThanOrEqual(p.n)
	case EQL:
		return n.Equal(p.n)
	case NEQL:
		return !n.Equal(p.n)
	default:
		// We should never hit this code path
		panic(fmt.Sprintf("p.op (%v) is not a valid comparison operator", p.op))
	}
}

var _ = rql.NumericPredicate(&numeric{})

func NumericValue(op ComparisonOp, n decimal.Decimal) rql.ValuePredicate {
	return &numericValue{numeric{
		op: op,
		n:  n,
	}}
}

type numericValue struct {
	numeric
}

func (p *numericValue) Marshal() interface{} {
	return []interface{}{"number", p.numeric.Marshal()}
}

func (p *numericValue) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("number"))(input) {
		return errz.MatchErrorf("must be formatted as ['number', <numeric_predicate>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as ['number', <numeric_predicate>]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as ['number', <numeric_predicate>] (missing the numeric predicate)")
	}
	if err := p.numeric.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("error unmarshalling the numeric predicate: %w", err)
	}
	return nil
}

func (p *numericValue) EvalValue(v interface{}) bool {
	n, ok := v.(float64)
	return ok && p.EvalNumeric(decimal.NewFromFloat(n))
}

var _ = rql.ValuePredicate(&numericValue{})
