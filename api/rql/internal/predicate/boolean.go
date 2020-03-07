package predicate

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
)

func Boolean(val bool) rql.ValuePredicate {
	p := &boolean{
		val: val,
	}
	p.primitiveValueBase = newPrimitiveValue(p)
	return p
}

type boolean struct {
	primitiveValueBase
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

func (p *boolean) EvalEntry(rql.Entry) bool {
	return p.val
}

func (p *boolean) EvalEntrySchema(*rql.EntrySchema) bool {
	return p.val
}

func (p *boolean) IsPrimary() bool {
	return true
}

var _ = meta.ValuePredicate(&boolean{})
var _ = rql.EntryPredicate(&boolean{})
var _ = rql.EntrySchemaPredicate(&boolean{})
