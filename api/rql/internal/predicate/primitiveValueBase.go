package predicate

import "github.com/puppetlabs/wash/api/rql/internal/primary/meta"

type primitiveValueBase struct {
	*meta.ValuePredicateBase
}

func newPrimitiveValue(p meta.ValuePredicate) primitiveValueBase {
	return primitiveValueBase{
		ValuePredicateBase: meta.NewValuePredicate(p),
	}
}

func (p *primitiveValueBase) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	return meta.MakeSchemaPredicate(svs.EndsWithPrimitiveValue())
}
