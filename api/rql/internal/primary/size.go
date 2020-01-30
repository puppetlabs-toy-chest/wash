package primary

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
)

func Size(p rql.NumericPredicate) rql.Primary {
	return predicate.Size(p).(rql.Primary)
}
