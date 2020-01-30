package ast

import (
	"encoding/json"

	"github.com/puppetlabs/wash/api/rql"
)

// MarshalJSON marshals the node into JSON
func MarshalJSON(n rql.ASTNode) ([]byte, error) {
	return json.Marshal(n.Marshal())
}

// UnmarshalJSON unmarshals the node from json
func UnmarshalJSON(b []byte, n rql.ASTNode) error {
	var input interface{}
	if err := json.Unmarshal(b, &input); err != nil {
		return err
	}
	return n.Unmarshal(input)
}
