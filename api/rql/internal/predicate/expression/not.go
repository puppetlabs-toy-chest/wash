package expression

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

func Not(p rql.ASTNode) rql.ASTNode {
	return &not{
		p: toAtom(p),
	}
}

type not struct {
	base
	p rql.ASTNode
}

func (n *not) Marshal() interface{} {
	return []interface{}{"NOT", n.p.Marshal()}
}

func (n *not) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("NOT"))(input) {
		return errz.MatchErrorf("must be formatted as [\"NOT\", <pe>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"NOT\", <pe>]")
	}
	if len(array) != 2 {
		return fmt.Errorf("NOT: missing the expression")
	}
	if err := n.p.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("NOT: error unmarshaling expression: %w", err)
	}
	n.p = unravelNTN(n.p)
	return nil
}

func (n *not) EvalEntry(e rql.Entry) bool {
	return !n.p.(rql.EntryPredicate).EvalEntry(e)
}

func (n *not) EvalEntrySchema(s *rql.EntrySchema) bool {
	return !n.p.(rql.EntrySchemaPredicate).EvalEntrySchema(s)
}

func (n *not) EvalValue(v interface{}) bool {
	return !n.p.(rql.ValuePredicate).EvalValue(v)
}

func (n *not) EvalString(str string) bool {
	return !n.p.(rql.StringPredicate).EvalString(str)
}

func (n *not) EvalNumeric(x decimal.Decimal) bool {
	return !n.p.(rql.NumericPredicate).EvalNumeric(x)
}

func (n *not) EvalTime(t time.Time) bool {
	return !n.p.(rql.TimePredicate).EvalTime(t)
}

func (n *not) EvalAction(action plugin.Action) bool {
	return !n.p.(rql.ActionPredicate).EvalAction(action)
}

var _ = expressionNode(&not{})
var _ = rql.EntryPredicate(&not{})
var _ = rql.EntrySchemaPredicate(&not{})
var _ = rql.ValuePredicate(&not{})
var _ = rql.StringPredicate(&not{})
var _ = rql.NumericPredicate(&not{})
var _ = rql.TimePredicate(&not{})
var _ = rql.ActionPredicate(&not{})
