package meta

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type ValueSchemaTestSuite struct {
	suite.Suite
}

func (suite *ValueSchemaTestSuite) TestNewSchema() {
	var rawSchema *plugin.JSONSchema
	// TODO: Once https://github.com/alecthomas/jsonschema/issues/40
	// is (properly) resolved, we should dynamically generate the
	// schema from a struct so maintainers can see what our mock looks
	// like. Right now, the (hacky) fix in our jsonschema fork generates
	// duplicate definitions for anonymous structs (and this behavior's
	// unpredictable), so we store the JSON in a fixture. Note that
	// it still generates the right schema, there's just some redundancy
	// in the generated schema.
	readSchemaFixture(suite.Suite, "before_munging", &rawSchema)
	var expected map[string]interface{}
	readSchemaFixture(suite.Suite, "after_munging", &expected)

	schema := NewValueSchema(rawSchema)

	actualBytes, err := json.Marshal(schema.loader.JsonSource())
	if err != nil {
		suite.FailNow("Failed to marshal the munged JSON schema: %v", err)
	}
	var actual map[string]interface{}
	if err := json.Unmarshal(actualBytes, &actual); err != nil {
		suite.FailNow("Failed to unmarshal the munged JSON schema: %v", err)
	}

	suite.Equal(expected, actual)
}

func (s *ValueSchemaTestSuite) TestSupports() {
	var rawSchema *plugin.JSONSchema
	readSchemaFixture(s.Suite, "before_munging", &rawSchema)
	schema := NewValueSchema(rawSchema)

	// Test valid value schemas
	svs := (NewSatisfyingValueSchema()).
		AddObject("dp").
		AddObject("dcp").
		AddObject("dcap").
		EndsWithPrimitiveValue()
	s.True(schema.Supports(svs))

	svs = (NewSatisfyingValueSchema()).
		AddObject("cp").
		EndsWithArray()
	s.True(schema.Supports(svs))

	svs = (NewSatisfyingValueSchema()).
		AddObject("dp").
		EndsWithAnything()
	s.True(schema.Supports(svs))

	// Now test invalid value schemas

	// "DP" is the invalid value here with the invalid property
	// "Foo"
	svs = (NewSatisfyingValueSchema()).
		AddObject("dp").
		AddObject("foo").
		EndsWithPrimitiveValue()
	s.False(schema.Supports(svs))

	// "AP" is a primitive type, so its value must be "null".
	// Here, however, it is an object.
	svs = (NewSatisfyingValueSchema()).
		AddObject("ap").
		EndsWithObject()
	s.False(schema.Supports(svs))

	// "DDP" is not a valid property of "DCP"
	svs = (NewSatisfyingValueSchema()).
		AddObject("dp").
		AddObject("dcp").
		AddObject("ddp").
		EndsWithAnything()
	s.False(schema.Supports(svs))
}

func TestValueSchema(t *testing.T) {
	suite.Run(t, new(ValueSchemaTestSuite))
}

// v is the value that the schema-fixture will be marshaled into.
// We keep it generic in case v is a map[string]interface{} object
// instead of a *plugin.JSONSchema value
func readSchemaFixture(s suite.Suite, name string, v interface{}) {
	filePath := path.Join("testdata", name+".json")
	rawSchema, err := ioutil.ReadFile(filePath)
	if err != nil {
		s.T().Fatal(fmt.Sprintf("Failed to read %v", filePath))
	}
	if err := json.Unmarshal(rawSchema, v); err != nil {
		s.T().Fatal(fmt.Sprintf("Failed to unmarshal %v: %v", filePath, err))
	}
}
