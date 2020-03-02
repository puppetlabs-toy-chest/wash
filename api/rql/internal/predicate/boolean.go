package predicate

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
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
	return []interface{}{"boolean", p.val}
}

func (p *boolean) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("boolean"))(input) {
		return errz.MatchErrorf("must be formatted as ['boolean', <value>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as ['boolean', <value>]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as ['boolean', <value>] (missing the value)")
	}
	val, ok := array[1].(bool)
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
