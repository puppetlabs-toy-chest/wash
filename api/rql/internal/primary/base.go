package primary

import (
	"fmt"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/matcher"
)

// Captures the common structure of [<primary_name>, <predicate>]
// found in most of the primaries
type base struct {
	name string
	// ptype => predicateType
	ptype string
	p     rql.ASTNode
}

func (p *base) Marshal() interface{} {
	return []interface{}{p.name, p.p.Marshal()}
}

func (p *base) Unmarshal(input interface{}) error {
	if !matcher.Array(matcher.Value(p.name))(input) {
		return errz.MatchErrorf("must be formatted as ['%v', <%v_predicate>]", p.name, p.ptype)
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf("must be formatted as ['%v', <%v_predicate>]", p.name, p.ptype)
	}
	if len(array) < 2 {
		return fmt.Errorf("missing the %v predicate", p.ptype)
	}
	if err := p.p.Unmarshal(array[1]); err != nil {
		// TODO: Make this a structured error
		return fmt.Errorf("%w", err)
	}
	return nil
}

func (p *base) EntryInDomain(rql.Entry) bool {
	return true
}

func (p *base) EntrySchemaInDomain(*rql.EntrySchema) bool {
	return true
}
