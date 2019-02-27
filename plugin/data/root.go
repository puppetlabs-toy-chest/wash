package data

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Root of the YAML plugin adapter
type Root struct {
	listableEntry `yaml:",inline"`
}

// Init does nothing.
func (r *Root) Init() error {
	return nil
}

// NewRoot loads the plugin description at path and returns a new plugin root.
func NewRoot(path string) (*Root, error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var plugin Root
	if err := yaml.Unmarshal(input, &plugin); err != nil {
		return nil, err
	}
	return &plugin, nil
}
