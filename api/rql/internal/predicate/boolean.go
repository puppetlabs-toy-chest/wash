package predicate

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
)

func Boolean(val bool) rql.ValuePredicate {
	return &boolean{
		val: val,
	}
}

type boolean struct {
	val bool
}

func (p *boolean) Marshal() interface{} {
	return p.val
}

func (p *boolean) Unmarshal(input interface{}) error {
	val, ok := input.(bool)
	if !ok {
		return errz.MatchErrorf("%v is not a valid Boolean value. Valid Boolean values are true, false", input)
	}
	p.val = val
	return nil
}

func (p *boolean) EvalValue(v interface{}) bool {
	val, ok := v.(bool)
	return ok && val == p.val
}

var _ = rql.ValuePredicate(&boolean{})
