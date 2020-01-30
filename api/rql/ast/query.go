package ast

import (
	"github.com/puppetlabs/wash/api/rql"
)

// Query returns an AST node representing an RQL query
func Query() rql.Primary {
	return PE_Primary()
}
