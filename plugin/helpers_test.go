package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToMetadata(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected MetadataMap
	}{
		{[]byte(`{"hello": [1, 2, 3]}`), MetadataMap{"hello": []interface{}{1.0, 2.0, 3.0}}},
		{struct {
			Name  string
			Value []int
		}{"me", []int{1, 2, 3}}, MetadataMap{"Name": "me", "Value": []interface{}{1.0, 2.0, 3.0}}},
	}
	for _, c := range cases {
		actual := ToMetadata(c.input)
		assert.Equal(t, c.expected, actual)
	}
}
