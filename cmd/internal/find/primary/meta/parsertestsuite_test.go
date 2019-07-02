package meta

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/ekinanp/jsonschema"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

// parserTestSuite represents a type that tests meta primary predicate parsers.
// It is a wrapper around the main parsertest.Suite type that includes additional
// support for schemaP testing via the key sequence helpers.
type parserTestSuite struct {
	parsertest.Suite
}

func (s *parserTestSuite) SetParser(parser predicate.Parser) {
	s.Parser = parser
	s.SchemaPParser = predicate.ToParser(func(tokens []string) (predicate.Predicate, []string, error) {
		p, tokens, err := s.Parser.Parse(tokens)
		if err != nil {
			return p, tokens, err
		}
		schemaP := p.(Predicate).schemaP()
		return schemaP, tokens, err
	})
}

// RSTC => RunSchemaTestCase. Saves some typing.
func (s *parserTestSuite) RSTC(input string, remInput string, trueKS string, falseKS ...string) {
	s.Suite.RSTC(input, remInput, s.newSchema(trueKS))
	if len(falseKS) > 0 {
		s.RNSTC(input, remInput, falseKS[0])
	}
}

// RNSTC => RunNegativeSchemaTestCase. Saves some typing
func (s *parserTestSuite) RNSTC(input string, remInput string, falseKS string) {
	s.Suite.RNSTC(input, remInput, s.newSchema(falseKS))
}

// ksStr should be something like ".key1[].key2 a", where the "a"
// indicates that the key sequence ends with an array. Similarly,
// "p" and "o" indicate that the key sequence ends with a primitive
// value/object, respectively.
func (s *parserTestSuite) parseKS(str string) keySequence {
	failNow := func(format string, a ...interface{}) {
		format = fmt.Sprintf("invalid key sequence %v: %v", str, format)
		msg := fmt.Sprintf(format, a...)
		panic(msg)
	}
	endValueTypeErrMsg := "no end value type was provided. valid end values are 'o', 'p', or 'a'"

	// The keyRegex in objectPredicate allows spaces. This doesn't.
	keyRegex := regexp.MustCompile(`^([^\.\[\] ]+)`)

	var parseKS func(s string) keySequence
	parseKS = func(s string) keySequence {
		if len(s) == 0 {
			failNow(endValueTypeErrMsg)
		}
		switch s[0] {
		case ']':
			failNow("array segments must begin with '['")
		case '[':
			// Found an array segment
			s = s[1:]
			if len(s) == 0 || s[0] != ']' {
				failNow("array segments must end with ']'")
			}
			s = s[1:]
			ks := parseKS(s)
			return ks.AddArray()
		case '.':
			s = s[1:]
			loc := keyRegex.FindStringIndex(s)
			if loc == nil {
				failNow("expected a key after '.'")
			}
			key := s[loc[0]:loc[1]]
			ks := parseKS(s[loc[1]:])
			return ks.AddObject(key)
		default:
			// Reached the end. String should be " <endValueType>" or "<endValueType>"
			s := strings.TrimPrefix(s, " ")
			if len(s) == 0 {
				failNow(endValueTypeErrMsg)
			}
			ks := keySequence{}
			switch s[0] {
			case 'a':
				return ks.EndsWithArray()
			case 'o':
				return ks.EndsWithObject()
			case 'p':
				return ks.EndsWithPrimitiveValue()
			default:
				failNow(endValueTypeErrMsg)
			}
		}
		return keySequence{}
	}
	return parseKS(str)
}

func (s *parserTestSuite) toJSONSchema(ksStr string) *plugin.JSONSchema {
	var toJSONSchemaHelper func(json interface{}) *jsonschema.Type
	toJSONSchemaHelper = func(json interface{}) *jsonschema.Type {
		s := &jsonschema.Type{}
		switch t := json.(type) {
		case map[string]interface{}:
			s.Type = "object"
			s.Properties = make(map[string]*jsonschema.Type)
			for prop, v := range t {
				s.Properties[prop] = toJSONSchemaHelper(v)
				s.MinProperties = s.MinProperties + 1
				s.AdditionalProperties = []byte("false")
			}
			return s
		case []interface{}:
			s.Type = "array"
			if len(t) > 0 {
				s.Items = toJSONSchemaHelper(t[0])
				s.MinItems = 1
			}
			return s
		default:
			s.Type = "null"
			return s
		}
	}
	schema := toJSONSchemaHelper(s.parseKS(ksStr).toJSON())
	return &jsonschema.Schema{
		Type: schema,
	}
}

func (s *parserTestSuite) newSchema(ksStr string) schema {
	return newSchema(s.toJSONSchema(ksStr))
}

// Even though the KS helpers are test helpers, they're complex
// enough to warrant some sanity checks
type KSHelpersTestSuite struct {
	parserTestSuite
}

// Even though parseKS is a test-helper, it's complex enough that it
// warrants some sanity checks
func (s *KSHelpersTestSuite) TestParseKS() {
	ks := s.parseKS("o")
	s.Equal(map[string]interface{}{}, ks.toJSON())
	ks = s.parseKS("a")
	s.Equal([]interface{}{}, ks.toJSON())
	ks = s.parseKS("p")
	s.Equal(nil, ks.toJSON())

	ks = s.parseKS("[] p")
	s.Equal([]interface{}{nil}, ks.toJSON())
	ks = s.parseKS(".key1 p")
	s.Equal(map[string]interface{}{"KEY1": nil}, ks.toJSON())

	ks = s.parseKS(".key1[][].key2.key3 o")
	expected := map[string]interface{}{
		"KEY1": []interface{}{
			[]interface{}{
				map[string]interface{}{
					"KEY2": map[string]interface{}{
						"KEY3": map[string]interface{}{},
					},
				},
			},
		},
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KSHelpersTestSuite) TestToJSONSchema() {
	serialize := func(schema *plugin.JSONSchema) map[string]interface{} {
		rawBytes, err := json.Marshal(schema)
		if err != nil {
			s.FailNow("Failed to marshal the munged JSON schema: %v", err)
		}
		var mp map[string]interface{}
		if err := json.Unmarshal(rawBytes, &mp); err != nil {
			s.FailNow("Failed to unmarshal the munged JSON schema: %v", err)
		}
		return mp
	}

	schema := s.toJSONSchema("o")
	s.Equal(map[string]interface{}{"type": "object"}, serialize(schema))
	schema = s.toJSONSchema("a")
	s.Equal(map[string]interface{}{"type": "array"}, serialize(schema))
	schema = s.toJSONSchema("p")
	s.Equal(map[string]interface{}{"type": "null"}, serialize(schema))

	schema = s.toJSONSchema(".key1[][].key2.key3 o")
	expected := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"minProperties":        float64(1),
		"properties": map[string]interface{}{
			"KEY1": map[string]interface{}{
				"type":     "array",
				"minItems": float64(1),
				"items": map[string]interface{}{
					"type":     "array",
					"minItems": float64(1),
					"items": map[string]interface{}{
						"type":                 "object",
						"additionalProperties": false,
						"minProperties":        float64(1),
						"properties": map[string]interface{}{
							"KEY2": map[string]interface{}{
								"type":                 "object",
								"additionalProperties": false,
								"minProperties":        float64(1),
								"properties": map[string]interface{}{
									"KEY3": map[string]interface{}{
										"type": "object",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	s.Equal(expected, serialize(schema))
}

func TestParseKS(t *testing.T) {
	suite.Run(t, new(KSHelpersTestSuite))
}
