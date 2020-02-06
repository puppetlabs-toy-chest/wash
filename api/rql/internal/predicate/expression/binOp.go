package expression

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
)

// Note that each binOp implements its own Eval* methods so that we
// can take advantage of short-circuiting

type binOp struct {
	base
	op string
	p1 rql.ASTNode
	p2 rql.ASTNode
}

func (o *binOp) Marshal() interface{} {
	return []interface{}{o.op, o.p1.Marshal(), o.p2.Marshal()}
}

func (o *binOp) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value(o.op))(input) {
		return errz.MatchErrorf("must be formatted as [\"%v\", <pe>, <pe>]", o.op)
	}
	array := input.([]interface{})
	if len(array) > 3 {
		return fmt.Errorf("must be formatted as [\"%v\", <pe>, <pe>]", o.op)
	}
	if len(array) != 3 {
		return fmt.Errorf("%v: missing one or both of the LHS and RHS expressions", o.op)
	}
	if err := o.p1.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("%v: error unmarshaling LHS expression: %w", o.op, err)
	}
	if err := o.p2.Unmarshal(array[2]); err != nil {
		return fmt.Errorf("%v: error unmarshaling RHS expression: %w", o.op, err)
	}
	o.p1 = unravelNTN(o.p1)
	o.p2 = unravelNTN(o.p2)
	return nil
}
