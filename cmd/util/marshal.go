package cmdutil

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
)

const (
	// JSON represents the json marshaller
	JSON = "json"
	// YAML represents the YAML marshaller
	YAML = "yaml"
	// TEXT represents an easily greppable text format
	TEXT = "text"
)

// Marshaller is a type that marshals a given value
type Marshaller func(interface{}) ([]byte, error)

// YamlMarshaler is a type that can be marshalled into
// YAML
type YamlMarshaler interface {
	MarshalYAML() ([]byte, error)
}

// TextMarshaler is a type that can be marshaled into
// Text output.
//
// For structured data, text output prefixes each line with the
// entire key path (using .KEY for map indexing and .N for array
// indexing) to make it more greppable. For example, given something
// like
//
//     AppArmorProfile:
//       Args:
//         - redis-server
//       Config:
//         AttachStdout: false
//         Cmd:
//           - redis-server
//
// Its Text output would be:
//
//     AppArmorProfile:
//     Args.0: redis-server
//     Config.AttachStdout: false
//     Config.Cmd.0: redis-server
//
type TextMarshaler interface {
	MarshalTEXT() ([]byte, error)
}

// NewMarshaller returns a marshaller that marshals values
// into the specified format. Currently only JSON or YAML
// are supported.
//
// All non-JSON marshallers have a default implementation
// of
//
//   "Marshal to JSON" =>
//   "Unmarshal the JSON" =>
//   "Marshal to <format>"
//
// (the first two steps are effectively "Marshal to a
// a JSON [map/array/value] Go type" so that people don't have
// to implement multiple Marshaler interfaces).
//
// For types that have their own MarshalJSON implementation,
// this could be a problem because the "Unmarshal the JSON"
// step unmarshals the data into a different type (so that
// the custom MarshalJSON implementation is not called).
// Thus, these types may need to implement the format-specific
// Marshaler interfaces:
//   * For YAML, this is cmdutil.YamlMarshaler
//   * For TEXT, this is cmdutil.TextMarshaler
//
func NewMarshaller(format string) (Marshaller, error) {
	switch format {
	case JSON:
		return Marshaller(func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		}), nil
	case YAML:
		return Marshaller(func(v interface{}) ([]byte, error) {
			switch t := v.(type) {
			case YamlMarshaler:
				return t.MarshalYAML()
			default:
				// yaml.Marshal marshals v to JSON then converts that JSON to YAML.
				return yaml.Marshal(v)
			}
		}), nil
	case TEXT:
		return Marshaller(toText), nil
	default:
		return nil, fmt.Errorf("the %v format is not supported. Supported formats are 'json' or 'yaml'", format)
	}
}

// Marshal marshals the given value
func (m Marshaller) Marshal(v interface{}) (string, error) {
	bytes, err := m(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func toText(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case TextMarshaler:
		return t.MarshalTEXT()
	default:
		goType, err := marshalToJSONGoType(v)
		if err != nil {
			return nil, err
		}
		s, e := textBuilder(goType, "")
		return []byte(s), e
	}
}

type keyValue struct {
	key   string
	value interface{}
}

func textBuilder(v interface{}, prefix string) (string, error) {
	// See https://golang.org/pkg/encoding/json/#Unmarshal for expected types.
	switch val := v.(type) {
	case map[string]interface{}:
		if prefix != "" {
			prefix += "."
		}
		ordered := make([]keyValue, 0, len(val))
		for k, v := range val {
			ordered = append(ordered, keyValue{k, v})
		}
		sort.Slice(ordered, func(i, j int) bool { return ordered[i].key < ordered[j].key })

		parts := make([]string, len(ordered))
		for i, kv := range ordered {
			more, err := textBuilder(kv.value, prefix+kv.key)
			if err != nil {
				return "", err
			}
			parts[i] = more
		}
		return strings.Join(parts, "\n"), nil
	case []interface{}:
		if prefix != "" {
			prefix += "."
		}
		parts := make([]string, len(val))
		for i, v := range val {
			more, err := textBuilder(v, prefix+strconv.Itoa(i))
			if err != nil {
				return "", err
			}
			parts[i] = more
		}
		return strings.Join(parts, "\n"), nil
	default:
		return fmt.Sprintf(prefix+": %v", val), nil
	}
}

// return value should be a map/array/primitive value type
func marshalToJSONGoType(v interface{}) (interface{}, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var goType interface{}
	err = json.Unmarshal(jsonBytes, &goType)
	return goType, err
}
