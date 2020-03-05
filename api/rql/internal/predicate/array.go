package predicate

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
)

func Array() rql.ValuePredicate {
	e := &arrayElement{p: NPE_ValuePredicate()}
	e.ValuePredicateBase = meta.NewValuePredicate(e)
	return &array{collectionBase{
		ctype:            "array",
		elementPredicate: e,
	}}
}

type array struct {
	collectionBase
}

var _ = meta.ValuePredicate(&array{})

type arrayElement struct {
	*meta.ValuePredicateBase
	selector interface{}
	p        meta.ValuePredicate
}

func (p *arrayElement) Marshal() interface{} {
	var marshalledSelector interface{}
	switch t := p.selector.(type) {
	case stringSelector:
		marshalledSelector = stringSelectorToStringMap[t]
	case int:
		marshalledSelector = float64(t)
	default:
		// Should never hit this code-path
		panic(fmt.Sprintf("Unknown selector %T", p.selector))
	}
	return []interface{}{marshalledSelector, p.p.Marshal()}
}

func (p *arrayElement) Unmarshal(input interface{}) error {
	array, ok := input.([]interface{})
	formatErrMsg := "element predicate: must be formatted as [<element_selector>, NPE ValuePredicate]"
	if !ok || len(array) < 1 {
		return errz.MatchErrorf(formatErrMsg)
	}
	if firstElem, ok := array[0].(string); ok {
		if firstElem != "some" && firstElem != "all" {
			return errz.MatchErrorf(formatErrMsg)
		}
		if firstElem == "some" {
			p.selector = some
		} else {
			p.selector = all
		}
	} else {
		firstElem, ok := array[0].(float64)
		if !ok {
			return errz.MatchErrorf(formatErrMsg)
		}
		if firstElem < 0 {
			return fmt.Errorf("element predicate: array index must be an unsigned integer (> 0)")
		}
		p.selector = int(firstElem)
	}
	if len(array) > 2 {
		return fmt.Errorf(formatErrMsg)
	} else if len(array) < 2 {
		return fmt.Errorf("%v (missing NPE ValuePredicate)", formatErrMsg)
	}
	if err := p.p.Unmarshal(array[1]); err != nil {
		return fmt.Errorf("element predicate: error unmarshalling the NPE ValuePredicate: %w", err)
	}
	return nil
}

func (p *arrayElement) EvalValue(v interface{}) bool {
	array, ok := v.([]interface{})
	if !ok {
		return false
	}
	switch t := p.selector.(type) {
	case int:
		return p.p.EvalValue(array[t])
	case stringSelector:
		switch t {
		case some:
			for _, v := range array {
				if p.p.EvalValue(v) {
					return true
				}
			}
			return false
		case all:
			for _, v := range array {
				if !p.p.EvalValue(v) {
					return false
				}
			}
			return true
		default:
			// Should never hit this code path
			panic(fmt.Sprintf("Unknown string selector %v", t))
		}
	default:
		// Should never hit this code path
		panic(fmt.Sprintf("Unknown selector %T", p.selector))
	}
}

func (p *arrayElement) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	return p.p.SchemaPredicate(svs.AddArray())
}

type stringSelector int8

const (
	some stringSelector = iota
	all
)

var stringSelectorToStringMap = map[stringSelector]string{
	some: "some",
	all:  "all",
}

var _ = meta.ValuePredicate(&arrayElement{})
