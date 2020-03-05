package meta

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ekinanp/jsonschema"
	"github.com/puppetlabs/wash/plugin"
	"github.com/xeipuuv/gojsonschema"
)

// ValueSchema represents a metadata value schema. It is a wrapper for
// plugin.JSONSchema.
type ValueSchema struct {
	loader gojsonschema.JSONLoader
}

func NewValueSchema(s *plugin.JSONSchema) ValueSchema {
	// Given a SVS like ".key1.key2 nil", the schema predicate's job is to use a JSON
	// schema validator to check that s supports the SVS' representative value of
	// {"key1": {"key2": nil}}. To do this correctly, we need to munge s a bit before
	// returning our schema instance. That's what this code does.
	var mungeProperties func(map[string]*jsonschema.Type) map[string]*jsonschema.Type
	var mungeType func(*jsonschema.Type)
	mungeProperties = func(properties map[string]*jsonschema.Type) map[string]*jsonschema.Type {
		// The meta primary searches for the first matching key where a matching key is the
		// first key s.t. upcase(matching_key) == upcase(key). Thus, all property names need
		// to be capitalized.
		upcasedProperties := make(map[string]*jsonschema.Type)
		for property, schema := range properties {
			p := strings.ToUpper(property)
			if _, ok := upcasedProperties[p]; ok {
				continue
			}
			mungeType(schema)
			upcasedProperties[p] = schema
		}
		return upcasedProperties
	}
	mungeType = func(t *jsonschema.Type) {
		if t == nil || len(t.Ref) > 0 {
			return
		}

		mungeType(t.Not)

		switch t.Type {
		case "array":
			mungeType(t.Items)
			mungeType(t.AdditionalItems)
			t.MinItems = 0
			for _, items := range [][]*jsonschema.Type{t.AllOf, t.AnyOf, t.OneOf} {
				for _, schema := range items {
					mungeType(schema)
				}
			}
		case "object":
			// Value schemas should be simple, so we shouldn't have to worry
			// about dependencies (for now).
			t.Dependencies = nil
			t.Properties = mungeProperties(t.Properties)
			t.PatternProperties = mungeProperties(t.PatternProperties)
			t.MinProperties = 0
			t.Required = nil
		default:
			// We've hit a primitive type. Normalize it by setting it to "null".
			t.Type = "null"
		}
	}

	mungeType(s.Type)
	for _, schema := range s.Definitions {
		mungeType(schema)
	}
	return ValueSchema{
		loader: gojsonschema.NewGoLoader(s),
	}
}

func (s ValueSchema) Supports(svs SatisfyingValueSchema) bool {
	if !svs.isComplete() {
		panic(fmt.Sprintf("svs#IsContainedIn: called on an incomplete SatisfyingValueSchema %T", svs))
	}
	for _, value := range svs.representativeValues {
		r, err := gojsonschema.Validate(s.loader, gojsonschema.NewGoLoader(value))
		if err != nil {
			msg := fmt.Sprintf("schema.Validate: gojsonschema.Validate: returned an unexpected error: %v", err)
			panic(msg)
		}
		if r.Valid() {
			return true
		}
	}
	return false
}

// String() is implemented to make test-failure output more readable
func (s ValueSchema) String() string {
	bytes, err := json.Marshal(s.loader.JsonSource())
	if err != nil {
		msg := fmt.Sprintf("s.String(): unexpected error marshalling the value schema: %v", err)
		panic(msg)
	}
	return string(bytes)
}
