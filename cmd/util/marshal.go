package cmdutil

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
)

const (
	// JSON represents the json marshaller
	JSON = "json"
	// YAML represents the YAML marshaller
	YAML = "yaml"
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
			// Use a JSONToYAML style encoding so that objects do not
			// have to implement multiple Marshal* interfaces.
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				return nil, err
			}
			return yaml.JSONToYAML(jsonBytes)
		}), nil
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
