package plugin

import (
	"github.com/ekinanp/jsonschema"
)

// JSONSchema represents a JSON schema
type JSONSchema = jsonschema.Schema

// TimeSchema represents the schema of a time.Time object
func TimeSchema() *JSONSchema {
	return jsonTypeToSchema(jsonschema.TimeType)
}

// IntegerSchema represents an integer's schema (int)
func IntegerSchema() *JSONSchema {
	return jsonTypeToSchema(jsonschema.IntegerType)
}

// NumberSchema represents a number's schema (float64)
func NumberSchema() *JSONSchema {
	return jsonTypeToSchema(jsonschema.NumberType)
}

// BooleanSchema represents a boolean's schema (bool)
func BooleanSchema() *JSONSchema {
	return jsonTypeToSchema(jsonschema.BoolType)
}

// StringSchema represents a string's schema (string)
func StringSchema() *JSONSchema {
	return jsonTypeToSchema(jsonschema.StringType)
}

func jsonTypeToSchema(t *jsonschema.Type) *JSONSchema {
	return &jsonschema.Schema{
		Type: t,
	}
}
