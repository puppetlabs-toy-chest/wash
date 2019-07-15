package apitypes

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
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
			"A": []string{"B", "C"},
			"B": []string{"D", "E"},
			"C": []string{"C", "F"},
			"D": []string{},
			"E": []string{"A", "C"},
			"F": []string{},
		}
		suite.Equal(expected, s.ToMap())

		// Ensure that duplicates are properly handled
		ABEC := suite.findNestedChild(s, "B", "E", "C")
		AC := suite.findNestedChild(s, "C")
		suite.True(ABEC == AC, "ABEC != AC")
		A := suite.findNestedChild(s)
		ABEA := suite.findNestedChild(A, "B", "E", "A")
		suite.True(A == ABEA, "A != ABEA")
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

// This should be called after suite.schema.FillChildren() is called
func (suite *EntrySchemaTestSuite) findNestedChild(s *EntrySchema, segments ...string) *EntrySchema {
	var visitedSegments []string
	child := s
	for _, segment := range segments {
		visitedSegments = append(visitedSegments, segment)
		child = child.GetChild(segment)
		if child == nil {
			suite.T().Fatal(fmt.Sprintf("Child %v does not exist", strings.Join(visitedSegments, "/")))
		}
	}
	return child
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
