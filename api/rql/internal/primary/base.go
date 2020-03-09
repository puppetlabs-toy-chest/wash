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
	// notNegatable is least common, so name the variable that
	notNegatable bool
	p            rql.ASTNode
}

func (p *base) Marshal() interface{} {
	return []interface{}{p.name, p.p.Marshal()}
}

func (p *base) Unmarshal(input interface{}) error {
	exprType := "NPE"
	if p.notNegatable {
		exprType = "PE"
	}
	errMsgPrefix := fmt.Sprintf("%v: must be formatted as [\"%v\", %v %vPredicate]", p.name, p.name, exprType, p.ptype)
	if !matcher.Array(matcher.Value(p.name))(input) {
		return errz.MatchErrorf(errMsgPrefix)
	}
	array := input.([]interface{})
	if len(array) > 2 {
		return fmt.Errorf(errMsgPrefix)
	}
	if len(array) < 2 {
		return fmt.Errorf("%v (missing %v %vPredicate)", errMsgPrefix, exprType, p.ptype)
	}
	if err := p.p.Unmarshal(array[1]); err != nil {
		// TODO: Make this a structured error
		return fmt.Errorf("%v: error unmarshalling the %v %vPredicate: %w", p.name, exprType, p.ptype, err)
	}
	return nil
}

func (p *base) IsPrimary() bool {
	return true
}
