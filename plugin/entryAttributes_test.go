package plugin

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type EntryAttributesTestSuite struct {
	suite.Suite
}

func (suite *EntryAttributesTestSuite) TestToMeta() {
	cases := []struct {
		input    interface{}
		expected EntryMetadata
	}{
		{[]byte(`{"hello": [1, 2, 3]}`), EntryMetadata{"hello": []interface{}{1.0, 2.0, 3.0}}},
		{struct {
			Name  string
			Value []int
		}{"me", []int{1, 2, 3}}, EntryMetadata{"Name": "me", "Value": []interface{}{1.0, 2.0, 3.0}}},
	}
	for _, c := range cases {
		actual := ToMeta(c.input)
		suite.Equal(c.expected, actual)
	}
}

// Easier to have a single test for everything since they're
// all redundant
func (suite *EntryAttributesTestSuite) TestEntryAttributes() {
	attr := EntryAttributes{}

	suite.Equal(false, attr.HasAtime())
	t := time.Now()
	attr.SetAtime(t)
	suite.Equal(t, attr.Atime())
	suite.Equal(true, attr.HasAtime())

	suite.Equal(false, attr.HasMtime())
	t = time.Now()
	attr.SetMtime(t)
	suite.Equal(t, attr.Mtime())
	suite.Equal(true, attr.HasMtime())

	suite.Equal(false, attr.HasCtime())
	t = time.Now()
	attr.SetCtime(t)
	suite.Equal(t, attr.Ctime())
	suite.Equal(true, attr.HasCtime())

	suite.Equal(false, attr.HasMode())
	m := os.FileMode(0777)
	attr.SetMode(m)
	suite.Equal(m, attr.Mode())
	suite.Equal(true, attr.HasMode())

	suite.Equal(false, attr.HasSize())
	sz := uint64(10)
	attr.SetSize(sz)
	suite.Equal(sz, attr.Size())
	suite.Equal(true, attr.HasSize())

	suite.Equal(EntryMetadata{}, attr.Meta())
	meta := EntryMetadata{"foo": "bar"}
	attr.SetMeta(meta)
	suite.Equal(meta, attr.Meta())
}

func TestEntryAttributes(t *testing.T) {
	suite.Run(t, new(EntryAttributesTestSuite))
}
