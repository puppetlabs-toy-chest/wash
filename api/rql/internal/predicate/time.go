package predicate

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
	"github.com/puppetlabs/wash/munge"
)

func Time(op ComparisonOp, t time.Time) rql.TimePredicate {
	return &tm{
		op: op,
		t:  t,
	}
}

// tm => time. Have to name it as such to avoid conflicting with
// the 'time' package
type tm struct {
	op ComparisonOp
	t  time.Time
}

func (p *tm) Marshal() interface{} {
	return []interface{}{string(p.op), p.t}
}

func (p *tm) Unmarshal(input interface{}) error {
	m := matcher.Array(func(v interface{}) bool {
		opStr, ok := v.(string)
		return ok && comparisonOpMap[ComparisonOp(opStr)]
	})
	if !m(input) {
		return errz.MatchErrorf("must be formatted as [<comparison_op>, <time>]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [<comparison_op>, <time>]")
	}
	if len(array) != 2 {
		return fmt.Errorf("must be formatted as [<comparison_op>, <time>] (missing the time)")
	}
	op := ComparisonOp(array[0].(string))
	if !comparisonOpMap[op] {
		return fmt.Errorf("%v is not a valid comparison op", op)
	}
	t, err := munge.ToTime(array[1])
	if err != nil {
		return err
	}
	p.op = op
	p.t = t
	return nil
}

func (p *tm) EvalTime(t time.Time) bool {
	switch p.op {
	case LT:
		return t.Before(p.t)
	case LTE:
		return t.Before(p.t) || t.Equal(p.t)
	case GT:
		return t.After(p.t)
	case GTE:
		return t.After(p.t) || t.Equal(p.t)
	case EQL:
		return t.Equal(p.t)
	default:
		// We should never hit this code path
		panic(fmt.Sprintf("p.op (%v) is not a valid comparison operator", p.op))
	}
}

var _ = rql.TimePredicate(&tm{})

func TimeValue(p rql.TimePredicate) rql.ValuePredicate {
	tm := &tmValue{
		TimePredicate: p,
	}
	tm.primitiveValueBase = newPrimitiveValue(tm)
	return tm
}

type tmValue struct {
	primitiveValueBase
	rql.TimePredicate
}

func (p *tmValue) Marshal() interface{} {
	return []interface{}{"time", p.TimePredicate.Marshal()}
}

func (p *tmValue) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value("time"))(input) {
		return errz.MatchErrorf("must be formatted as [\"time\", NPE TimePredicate]")
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as [\"time\", NPE TimePredicate]")
	}
	if len(array) < 2 {
		return fmt.Errorf("must be formatted as [\"time\", NPE TimePredicate] (missing the NPE TimePredicate)")
	}
	if err := p.TimePredicate.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("error unmarshalling the NPE TimePredicate: %w", err)
	}
	return nil
}

func (p *tmValue) EvalValue(v interface{}) bool {
	t, err := munge.ToTime(v)
	return err == nil && p.EvalTime(t)
}

var _ = meta.ValuePredicate(&tmValue{})
