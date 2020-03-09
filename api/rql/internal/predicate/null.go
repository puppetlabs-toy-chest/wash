package predicate

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
)

func Null() rql.ValuePredicate {
	n := &null{}
	n.primitiveValueBase = newPrimitiveValue(n)
	return n
}

type null struct {
	primitiveValueBase
}

func (p *null) Marshal() interface{} {
	return nil
}

func (p *null) Unmarshal(input interface{}) error {
	if input != nil {
		return errz.MatchErrorf("must be null")
	}
	return nil
}

func (p *null) EvalValue(v interface{}) bool {
	return v == nil
}

var _ = meta.ValuePredicate(&null{})
