package predicate

import (
	"fmt"
	"strings"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
)

func Object() rql.ValuePredicate {
	e := &objectElement{p: NPE_ValuePredicate()}
	e.ValuePredicateBase = meta.NewValuePredicate(e)
	return &object{collectionBase{
		ctype:            "object",
		elementPredicate: e,
	}}
}

type object struct {
	collectionBase
}

var _ = rql.ValuePredicate(&object{})

type objectElement struct {
	*meta.ValuePredicateBase
	key string
	p   meta.ValuePredicate
}

func (p *objectElement) Marshal() interface{} {
	return []interface{}{[]interface{}{"key", p.key}, p.p.Marshal()}
}

func (p *objectElement) Unmarshal(input interface{}) error {
	array, ok := input.([]interface{})
	formatErrMsg := "element predicate: must be formatted as [<element_selector>, NPE ValuePredicate]"
	if !ok || len(array) < 1 {
		return errz.MatchErrorf(formatErrMsg)
	}
	keySelector, ok := array[0].([]interface{})
	if !ok || len(keySelector) < 1 || keySelector[0] != "key" {
		return errz.MatchErrorf(formatErrMsg)
	}
	if len(keySelector) > 2 {
		return fmt.Errorf(formatErrMsg)
	}
	if len(keySelector) < 2 {
		return fmt.Errorf("element predicate: missing the key")
	}
	key, ok := keySelector[1].(string)
	if !ok {
		return fmt.Errorf("element predicate: key must be a string, not %T", keySelector[1])
	}
	p.key = key
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

func (p *objectElement) EvalValue(v interface{}) bool {
	obj, ok := v.(map[string]interface{})
	if !ok {
		return false
	}
	k, found := p.findMatchingKey(obj)
	return found && p.p.EvalValue(obj[k])
}

func (p *objectElement) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	return p.p.SchemaPredicate(svs.AddObject(p.key))
}

func (p *objectElement) findMatchingKey(mp map[string]interface{}) (string, bool) {
	upcasedKey := strings.ToUpper(p.key)
	for k := range mp {
		if strings.ToUpper(k) == upcasedKey {
			return k, true
		}
	}
	return "", false
}

var _ = meta.ValuePredicate(&objectElement{})
