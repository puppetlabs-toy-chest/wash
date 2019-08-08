package apitypes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	"github.com/stretchr/testify/suite"
)

type EntrySchemaTestSuite struct {
	suite.Suite
}

func (suite *EntrySchemaTestSuite) TestUnmarshalJSON_EmptySchema() {
	var s *EntrySchema
	suite.Regexp("non-empty.*JSON.*", json.Unmarshal([]byte("{}"), &s))
}

func (suite *EntrySchemaTestSuite) TestUnmarshalJSON_UnknownSchema() {
	var s *EntrySchema
	if err := json.Unmarshal([]byte("null"), &s); err != nil {
		msg := fmt.Sprintf("json.Unmarshal returned an unexpected error: %v", err)
		suite.FailNow(msg)
	}
	suite.Equal((*EntrySchema)(nil), s)
}

func (suite *EntrySchemaTestSuite) TestUnmarshalJSON_KnownSchema_ValidSchema() {
	s, err := suite.readFixture("validSchema")
	if suite.NoError(err) {
		expected := map[string][]string{
			"A":         []string{"A/B", "A/C"},
			"A/B":       []string{"A/B/D", "A/B/E"},
			"A/C":       []string{"A/C", "A/C/F"},
			"A/B/D":     []string{},
			"A/B/E":     []string{"A", "A/B/E/C"},
			"A/C/F":     []string{},
			"A/B/E/C":   []string{"A/B/E/C", "A/B/E/C/F"},
			"A/B/E/C/F": []string{},
		}
		suite.Equal(expected, s.ToMap())
	}
}

func (suite *EntrySchemaTestSuite) TestUnmarshalJSON_KnownSchema_InvalidSchema() {
	_, err := suite.readFixture("invalidSchema")
	// Should only report A's error since it's TypeID field is a number
	suite.Regexp("number", err)
}

func TestEntrySchema(t *testing.T) {
	suite.Run(t, new(EntrySchemaTestSuite))
}

func (suite *EntrySchemaTestSuite) readFixture(name string) (*EntrySchema, error) {
	filePath := path.Join("testdata", name+".json")
	rawSchema, err := ioutil.ReadFile(filePath)
	if err != nil {
		suite.FailNow(fmt.Sprintf("Failed to read %v", filePath))
		return nil, nil
	}
	var s *EntrySchema
	if err := json.Unmarshal(rawSchema, &s); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal %v: %v", filePath, err)
	}
	return s, nil
}
