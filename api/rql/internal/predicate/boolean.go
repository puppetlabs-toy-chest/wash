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
		return errz.MatchErrorf("must be formatted as <boolean_value>")
	}
	p.val = val
	return nil
}

func (p *boolean) ValueInDomain(v interface{}) bool {
	_, ok := v.(bool)
	return ok
}

func (p *boolean) EvalValue(v interface{}) bool {
	return v.(bool) == p.val
}

func (p *boolean) EntryInDomain(rql.Entry) bool {
	return true
}

func (p *boolean) EvalEntry(_ rql.Entry) bool {
	return p.val
}

func (p *boolean) EntrySchemaInDomain(*rql.EntrySchema) bool {
	return true
}

func (p *boolean) EvalEntrySchema(_ *rql.EntrySchema) bool {
	return p.val
}

var _ = rql.ValuePredicate(&boolean{})
var _ = rql.EntryPredicate(&boolean{})
var _ = rql.EntrySchemaPredicate(&boolean{})
