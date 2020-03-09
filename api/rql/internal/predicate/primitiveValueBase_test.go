package predicate

import (
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
)

// Contains common test code for the primitive types
//
// TODO: Might be possible to put some other tests here,
// but that is something to look into later.

type PrimitiveValueTestSuite struct {
	asttest.Suite
}

// VS => ValueSchemas
func (s *PrimitiveValueTestSuite) VS(valTypes ...string) []map[string]interface{} {
	schemas := []map[string]interface{}{}
	for _, valType := range valTypes {
		schemas = append(schemas, map[string]interface{}{"type": valType})
	}
	return schemas
}
