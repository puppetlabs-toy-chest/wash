package meta

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getlantern/deepcopy"

	"github.com/ekinanp/jsonschema"
	"github.com/puppetlabs/wash/plugin"
	"github.com/xeipuuv/gojsonschema"
)

// schema represents an entry metadata schema. It is a wrapper for
// plugin.JSONSchema.
type schema struct {
	schemaLoader gojsonschema.JSONLoader
	// existenceSchemaLoader is schemaLoader but with "minProperties"/"minItems"
	// set to 0. It is needed to correctly evaluate "-exists"'s schema predicate.
	// See the implementation of IsValidKeySequence to understand how this is used.
	//
	// NOTE: Omitting "minProperties"/"minItems" is not a good idea because that
	// stops us from optimizing the empty predicate. Optimizing the empty predicate
	// is important. Otherwise, accidental user input like "-m empty" has a chance
	// of generating many API requests because the corresponding schema predicate would
	// return true for things like Docker containers, EC2 instances, etc (since the
	// best it could do is check if the key sequence ends in an object/array, and entry
	// metadata is always a JSON object).
	//
	// NOTE: This hacky fix can be removed if/when we decide to walk the schema ourselves.
	existenceSchemaLoader gojsonschema.JSONLoader
}

func newSchema(s *plugin.JSONSchema) schema {
	// Given a meta primary key sequence like ".key1.key2 5", the schema predicate's
	// job is to check that m['key1']['key2'] == primitive_value. To do this correctly,
	// we need to munge s a bit before returning our schema instance. That's what this
	// code does.
	var mungeProperties func(map[string]*jsonschema.Type, bool) map[string]*jsonschema.Type
	var mungeType func(*jsonschema.Type, bool)
	var mungeSchema func(*plugin.JSONSchema, bool) gojsonschema.JSONLoader
	mungeProperties = func(properties map[string]*jsonschema.Type, forExistenceSchema bool) map[string]*jsonschema.Type {
		// The meta primary searches for the first matching key where a matching key is the
		// first key s.t. upcase(matching_key) == upcase(key). Thus, all property names need
		// to be capitalized.
		upcasedProperties := make(map[string]*jsonschema.Type)
		for property, schema := range properties {
			if _, ok := upcasedProperties[property]; ok {
				continue
			}
			mungeType(schema, forExistenceSchema)
			p := strings.ToUpper(property)
			upcasedProperties[p] = schema
		}
		return upcasedProperties
	}
	mungeType = func(t *jsonschema.Type, forExistenceSchema bool) {
		if t == nil || len(t.Ref) > 0 {
			return
		}

		mungeType(t.Not, forExistenceSchema)

		switch t.Type {
		case "array":
			mungeType(t.Items, forExistenceSchema)
			mungeType(t.AdditionalItems, forExistenceSchema)
			if forExistenceSchema {
				t.MinItems = 0
			}
			for _, items := range [][]*jsonschema.Type{t.AllOf, t.AnyOf, t.OneOf} {
				for _, schema := range items {
					mungeType(schema, forExistenceSchema)
				}
			}
		case "object":
			// Metadata schemas should be simple, so we shouldn't have to worry
			// about dependencies (for now).
			t.Dependencies = nil
			t.Properties = mungeProperties(t.Properties, forExistenceSchema)
			t.PatternProperties = mungeProperties(t.PatternProperties, forExistenceSchema)
			if forExistenceSchema {
				t.MinProperties = 0
			} else if t.Required != nil || t.MinProperties >= 1 {
				t.MinProperties = 1
			}
			t.Required = nil
		default:
			// We've hit a primitive type. Normalize it by setting it to "null".
			t.Type = "null"
		}
	}
	mungeSchema = func(s *plugin.JSONSchema, forExistenceSchema bool) gojsonschema.JSONLoader {
		mungeType(s.Type, forExistenceSchema)
		for _, schema := range s.Definitions {
			mungeType(schema, forExistenceSchema)
		}
		return gojsonschema.NewGoLoader(s)
	}

	var existenceSchema *plugin.JSONSchema
	if err := deepcopy.Copy(&existenceSchema, s); err != nil {
		msg := fmt.Sprintf("meta.newSchema: failed to deepcopy s: %v", err)
		panic(msg)
	}
	schemaLoader := mungeSchema(s, false)
	existenceSchemaLoader := mungeSchema(existenceSchema, true)

	return schema{
		schemaLoader:          schemaLoader,
		existenceSchemaLoader: existenceSchemaLoader,
	}
}

// IsValidKeySequence returns true if ks is a valid key sequence in s, false
// otherwise.
func (s schema) IsValidKeySequence(ks keySequence) bool {
	var r *gojsonschema.Result
	var err error
	validate := func(schema gojsonschema.JSONLoader, ks keySequence) bool {
		r, err = gojsonschema.Validate(schema, gojsonschema.NewGoLoader(ks.toJSON()))
		if err != nil {
			msg := fmt.Sprintf("s.Validate: gojsonschema.Validate: returned an unexpected error: %v", err)
			panic(msg)
		}
		return r.Valid()
	}
	if !ks.checkExistence {
		return validate(s.schemaLoader, ks)
	}
	// We're checking for ks' existence. This reduces to checking that the ks
	// ends with an object OR an array OR a primitive value since those are
	// the three possible JSON value types. Note that since the existence
	// schema sets "minProperties"/"minItems" to 0, it is enough for us to
	// pass-in an empty object/array. Otherwise, if our key sequence happens to
	// lead to e.g. an object and that object has "minProperties > 0", then we
	// would have had to include one of the specified properties in our key
	// sequence's JSON serialization. Figuring out that specified property would
	// require a schema walk, which we don't want.
	return validate(s.existenceSchemaLoader, ks.EndsWithObject()) ||
		validate(s.existenceSchemaLoader, ks.EndsWithArray()) ||
		validate(s.existenceSchemaLoader, ks.EndsWithPrimitiveValue())
}

// String() is implemented to make test-failure output more readable
func (s schema) String() string {
	bytes, err := json.Marshal(s.schemaLoader.JsonSource())
	if err != nil {
		msg := fmt.Sprintf("s.String(): unexpected error marshalling the schema: %v", err)
		panic(msg)
	}
	return string(bytes)
}
