package plugin

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
}

func (suite *HelpersTestSuite) TestToMetadata() {
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
		suite.Equal(c.expected, actual)
	}
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
