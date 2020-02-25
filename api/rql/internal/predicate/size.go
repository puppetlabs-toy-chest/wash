package predicate

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
	"github.com/shopspring/decimal"
)

// As a value predicate, Size is a predicate on the size
// of an object/array. As an entry predicate, Size is a
// predicate on the entry's size attribute.
func Size(p rql.NumericPredicate) rql.ValuePredicate {
	return &size{
		p: p,
	}
}

type size struct {
	p rql.NumericPredicate
}

func (p *size) Marshal() interface{} {
	return []interface{}{"size", p.p.Marshal()}
}

func (p *size) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("size"))(input) {
		return errz.MatchErrorf("size: must be formatted as [\"size\", PE NumericPredicate]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("size: must be formatted as [\"size\", PE NumericPredicate]")
	}
	if len(array) < 2 {
		return fmt.Errorf("size: must be formatted as [\"size\", PE NumericPredicate] (missing PE NumericPredicate)")
	}
	if err := p.p.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("size: error unmarshalling the PE NumericPredicate: %w", err)
	}
	return nil
}

func (p *size) EvalValue(v interface{}) bool {
	switch t := v.(type) {
	case map[string]interface{}:
		return p.p.EvalNumeric(decimal.NewFromInt(int64(len(t))))
	case []interface{}:
		return p.p.EvalNumeric(decimal.NewFromInt(int64(len(t))))
	default:
		return false
	}
}

func (p *size) IsPrimary() bool {
	return true
}

func (p *size) EvalEntry(e rql.Entry) bool {
	return p.p.EvalNumeric(decimal.NewFromInt(int64(e.Attributes.Size())))
}

var _ = rql.ValuePredicate(&size{})
var _ = rql.EntryPredicate(&size{})
