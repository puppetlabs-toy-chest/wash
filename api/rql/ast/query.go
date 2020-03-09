package ast

import (
	"github.com/puppetlabs/wash/api/rql"
)

// Query returns an AST node representing an RQL query
func Query() rql.Query {
	return PE_Primary()
}
