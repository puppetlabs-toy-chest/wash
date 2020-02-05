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

func (suite *EntryAttributesTestSuite) TestToJSONObject() {
	cases := []struct {
		input    interface{}
		expected JSONObject
	}{
		{[]byte(`{"hello": [1, 2, 3]}`), JSONObject{"hello": []interface{}{1.0, 2.0, 3.0}}},
		{struct {
			Name  string
			Value []int
		}{"me", []int{1, 2, 3}}, JSONObject{"Name": "me", "Value": []interface{}{1.0, 2.0, 3.0}}},
		{map[string]interface{}{"1": 2}, JSONObject{"1": 2}},
		{JSONObject{"1": 2}, JSONObject{"1": 2}},
	}
	for _, c := range cases {
		actual := ToJSONObject(c.input)
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
	doUnmarshalJSONTests := func() {
		attrJSON, err := json.Marshal(attr)
		suite.NoError(err)
		unmarshaledAttr := EntryAttributes{}
		err = json.Unmarshal(attrJSON, &unmarshaledAttr)
		if suite.NoError(err) {
			suite.Equal(attr, unmarshaledAttr)
		}
	}

	// ToMap - used for listing attributes - and JSON marshaling may have different representations.
	expectedMp := make(map[string]interface{})

	// Tests for Atime
	suite.Equal(false, attr.HasAtime())
	suite.Equal(expectedMp, attr.ToMap())
	t := timeNow()
	attr.SetAtime(t)
	expectedMp["atime"] = t
	suite.Equal(t, attr.Atime())
	suite.Equal(true, attr.HasAtime())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for Mtime
	suite.Equal(false, attr.HasMtime())
	suite.Equal(expectedMp, attr.ToMap())
	t = timeNow()
	attr.SetMtime(t)
	expectedMp["mtime"] = t
	suite.Equal(t, attr.Mtime())
	suite.Equal(true, attr.HasMtime())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for Ctime
	suite.Equal(false, attr.HasCtime())
	suite.Equal(expectedMp, attr.ToMap())
	t = timeNow()
	attr.SetCtime(t)
	expectedMp["ctime"] = t
	suite.Equal(t, attr.Ctime())
	suite.Equal(true, attr.HasCtime())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for Crtime
	suite.Equal(false, attr.HasCrtime())
	suite.Equal(expectedMp, attr.ToMap())
	t = timeNow()
	attr.SetCrtime(t)
	expectedMp["crtime"] = t
	suite.Equal(t, attr.Crtime())
	suite.Equal(true, attr.HasCrtime())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for OS
	suite.Equal(false, attr.HasOS())
	suite.Equal(expectedMp, attr.ToMap())
	o := OS{LoginShell: PowerShell}
	attr.SetOS(o)
	expectedMp["os"] = map[string]interface{}{"login_shell": "powershell"}
	suite.Equal(o, attr.OS())
	suite.Equal(true, attr.HasOS())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for Mode
	suite.Equal(false, attr.HasMode())
	suite.Equal(expectedMp, attr.ToMap())
	m := os.FileMode(0777 | os.ModeCharDevice | os.ModeDir)
	attr.SetMode(m)
	expectedMp["mode"] = m.String()
	suite.Equal(m, attr.Mode())
	suite.Equal(true, attr.HasMode())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()

	// Tests for Size
	suite.Equal(false, attr.HasSize())
	suite.Equal(expectedMp, attr.ToMap())
	sz := uint64(10)
	attr.SetSize(sz)
	expectedMp["size"] = sz
	suite.Equal(sz, attr.Size())
	suite.Equal(true, attr.HasSize())
	suite.Equal(expectedMp, attr.ToMap())
	doUnmarshalJSONTests()
}

func TestEntryAttributes(t *testing.T) {
	suite.Run(t, new(EntryAttributesTestSuite))
}
