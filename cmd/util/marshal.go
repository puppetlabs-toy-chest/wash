package cmdutil

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	goyaml "gopkg.in/yaml.v2"
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

// NewMarshaller returns a marshaller that marshals values
// into the specified format. Currently, only JSON or YAML
// are supported.
func NewMarshaller(format string) (Marshaller, error) {
	switch format {
	case JSON:
		return Marshaller(func(v interface{}) ([]byte, error) {
			return json.MarshalIndent(v, "", "  ")
		}), nil
	case YAML:
		return Marshaller(func(v interface{}) ([]byte, error) {
			switch t := v.(type) {
			case goyaml.Marshaler:
				return goyaml.Marshal(t)
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

// toText generates a textual representation of structured data that prefixes each line with the
// entire key path (using .KEY for map indexing and .N for array indexing) to make it more
// greppable. Uses textBuilder to recursively construct the output.
//
// Sample output:
//     AppArmorProfile:
//     Args.0: redis-server
//     Config.AttachStdout: false
//     Config.Cmd.0: redis-server
func toText(v interface{}) ([]byte, error) {
	s, e := textBuilder(v, "")
	return []byte(s), e
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
