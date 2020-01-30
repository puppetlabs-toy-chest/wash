package primary

import (
	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
)

func Boolean(val bool) rql.Primary {
	return predicate.Boolean(val).(rql.Primary)
}
