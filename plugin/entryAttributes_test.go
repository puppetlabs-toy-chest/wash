package plugin

import (
	"encoding/json"
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

// Easier to have a single test for everything since there's a lot
// of redundant, performance-optimized code.
func (suite *EntryAttributesTestSuite) TestEntryAttributes() {
	// timeNow returns the current time by marshalling time.Now() into JSON
	// and then unmarshalling it. We need this function for the unmarshal JSON
	// tests on the time attributes, because time.Now() includes the monotonic
	// clock + location information, both of which are lost when it is marshaled
	// into JSON. Hence without timeNow, the unmarshal JSON tests would always fail.
	timeNow := func() time.Time {
		t := time.Now()
		bytes, _ := json.Marshal(t)
		unmarshaledTime := time.Time{}
		_ = json.Unmarshal(bytes, &unmarshaledTime)
		return unmarshaledTime
	}
	attr := EntryAttributes{}
	attr.meta = EntryMetadata{}
	expectedMp := make(map[string]interface{})
	expectedMp["meta"] = EntryMetadata{}
	doUnmarshalJSONTests := func() {
		attrJSON, err := json.Marshal(expectedMp)
		if err != nil {
			panic("assertUnmarshalJSON: could not marshal expectedMp, which is a map[string]interface{} object")
		}
		unmarshaledAttr := EntryAttributes{}
		err = json.Unmarshal(attrJSON, &unmarshaledAttr)
		if suite.NoError(err) {
			suite.Equal(attr, unmarshaledAttr)
		}
	}

	// Tests for Atime
	suite.Equal(false, attr.HasAtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	t := timeNow()
	attr.SetAtime(t)
	expectedMp["atime"] = t
	suite.Equal(t, attr.Atime())
	suite.Equal(true, attr.HasAtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()

	// Tests for Mtime
	suite.Equal(false, attr.HasMtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	t = timeNow()
	attr.SetMtime(t)
	expectedMp["mtime"] = t
	suite.Equal(t, attr.Mtime())
	suite.Equal(true, attr.HasMtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()

	// Tests for Ctime
	suite.Equal(false, attr.HasCtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	t = timeNow()
	attr.SetCtime(t)
	expectedMp["ctime"] = t
	suite.Equal(t, attr.Ctime())
	suite.Equal(true, attr.HasCtime())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()

	// Tests for Mode
	suite.Equal(false, attr.HasMode())
	suite.Equal(expectedMp, attr.ToMap(true))
	m := os.FileMode(0777)
	attr.SetMode(m)
	expectedMp["mode"] = m
	suite.Equal(m, attr.Mode())
	suite.Equal(true, attr.HasMode())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()

	// Tests for Size
	suite.Equal(false, attr.HasSize())
	suite.Equal(expectedMp, attr.ToMap(true))
	sz := uint64(10)
	attr.SetSize(sz)
	expectedMp["size"] = sz
	suite.Equal(sz, attr.Size())
	suite.Equal(true, attr.HasSize())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()

	// Tests for Meta
	suite.Equal(EntryMetadata{}, attr.Meta())
	meta := EntryMetadata{"foo": "bar"}
	attr.SetMeta(meta)
	expectedMp["meta"] = meta
	suite.Equal(meta, attr.Meta())
	suite.Equal(expectedMp, attr.ToMap(true))
	doUnmarshalJSONTests()
}

func TestEntryAttributes(t *testing.T) {
	suite.Run(t, new(EntryAttributesTestSuite))
}
