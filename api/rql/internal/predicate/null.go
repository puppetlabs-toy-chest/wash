package predicate

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
)

func Null() rql.ValuePredicate {
	return &null{}
}

type null struct{}

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

var _ = rql.ValuePredicate(&null{})
