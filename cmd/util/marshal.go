package cmdutil

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
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
		return Marshaller(yaml.Marshal), nil
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
