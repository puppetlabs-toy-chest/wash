package types

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

type EntrySchemaTestSuite struct {
	suite.Suite
}

func (suite *EntrySchemaTestSuite) TestPrune_CanPruneRoot() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP()
	s = Prune(s, p, NewOptions())
	suite.Nil(s)
}

func (suite *EntrySchemaTestSuite) TestPrune_Tree() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP("A/B/D")
	s = Prune(s, p, NewOptions())
	expected := map[string][]string{
		"A":     []string{"A/B"},
		"A/B":   []string{"A/B/D"},
		"A/B/D": []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *EntrySchemaTestSuite) TestPrune_Complex() {
	s := suite.readFixture("complex")
	p := suite.makeSchemaP("A/B/D")
	s = Prune(s, p, NewOptions())
	expected := map[string][]string{
		"A":     []string{"A/B"},
		"A/B":   []string{"A/B/D", "A/B/E"},
		"A/B/D": []string{},
		"A/B/E": []string{"A"},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *EntrySchemaTestSuite) TestPrune_RealWorld_AWS() {
	s := suite.readFixture("real_world_aws")
	p := ToEntrySchemaP(func(s *EntrySchema) bool {
		for _, action := range s.Actions() {
			if action == "exec" {
				return true
			}
		}
		return false
	})
	s = Prune(s, p, NewOptions())
	expected := map[string][]string{
		"aws":                                 []string{"aws/profile"},
		"aws/profile":                         []string{"aws/profile/resources"},
		"aws/profile/resources":               []string{"aws/profile/resources/ec2"},
		"aws/profile/resources/ec2":           []string{"aws/profile/resources/ec2/instances"},
		"aws/profile/resources/ec2/instances": []string{"aws/profile/resources/ec2/instances/instance"},
		"aws/profile/resources/ec2/instances/instance": []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Tree() {
	s := suite.readFixture("tree")

	p := suite.makeSchemaP("A", "A/B", "A/C", "A/B/D", "A/B/E", "A/C/F", "A/C/G")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/C", "A/B/D", "A/B/E", "A/C/F", "A/C/G")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "A", "A/B", "A/C", "A/B/D", "A/B/E", "A/C/F", "A/C/G")

	p = suite.makeSchemaP("A/B/D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/B/D")
	suite.assertKeepResult(result, false, "A/B/E", "A/C", "A/C/F", "A/C/G")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Triangle() {
	s := suite.readFixture("triangle")

	p := suite.makeSchemaP("A", "A/B", "A/B/C")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/B/C")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "A", "A/B", "A/B/C")

	p = suite.makeSchemaP("A/B")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/B/C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Complex() {
	s := suite.readFixture("complex")

	p := suite.makeSchemaP(
		"A",
		"A/B",
		"A/C",
		"A/B/D",
		"A/B/E",
		"A/C/F",
		"A/B/E/C",
		"A/B/E/C/F",
	)
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(
		result,
		true,
		"A",
		"A/B",
		"A/C",
		"A/B/D",
		"A/B/E",
		"A/C/F",
		"A/B/E/C",
		"A/B/E/C/F",
	)

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(
		result,
		false,
		"A",
		"A/B",
		"A/C",
		"A/B/D",
		"A/B/E",
		"A/C/F",
		"A/B/E/C",
		"A/B/E/C/F",
	)

	p = suite.makeSchemaP("A/B/E/C/F")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/B/E", "A/B/E/C", "A/B/E/C/F")
	suite.assertKeepResult(result, false, "A/B/D", "A/C", "A/C/F")

	p = suite.makeSchemaP("A/B/D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "A/B", "A/B/D", "A/B/E")
	suite.assertKeepResult(result, false, "A/B/E/C", "A/B/E/C/F", "A/C", "A/C/F")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_DefaultsToPartialMetadataSchema() {
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("A", "object", "foo")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A")
	suite.assertKeepResult(result, false, "A/B", "A/C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_NilMetadataSchema_FallsbackToPartialMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("A", "object", "foo")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "A")
	suite.assertKeepResult(result, false, "A/B", "A/C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_SetToMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("A/B", "array")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "A", "A/B")
	suite.assertKeepResult(result, false, "A/C")
}

func TestEntrySchema(t *testing.T) {
	suite.Run(t, new(EntrySchemaTestSuite))
}

func (suite *EntrySchemaTestSuite) readFixture(name string) *EntrySchema {
	filePath := path.Join("testdata", name+".json")
	rawSchema, err := ioutil.ReadFile(filePath)
	if err != nil {
		suite.T().Fatal(fmt.Sprintf("Failed to read %v", filePath))
	}
	var s *apitypes.EntrySchema
	if err := json.Unmarshal(rawSchema, &s); err != nil {
		suite.T().Fatal(fmt.Sprintf("Failed to unmarshal %v: %v", filePath, err))
	}
	return s
}

func (suite *EntrySchemaTestSuite) makeSchemaP(trueValues ...string) EntrySchemaPredicate {
	return ToEntrySchemaP(func(s *EntrySchema) bool {
		for _, path := range trueValues {
			if s.Path() == path {
				return true
			}
		}
		return false
	})
}

func (suite *EntrySchemaTestSuite) makeMetadataSchemaP(trueValue string, expectedType string, expectedProperties ...string) EntrySchemaPredicate {
	return ToEntrySchemaP(func(s *EntrySchema) bool {
		if s.Path() != trueValue {
			return false
		}
		if s.MetadataSchema() == nil {
			return false
		}
		result := suite.Equal(s.MetadataSchema().Type.Type, expectedType)
		for _, property := range expectedProperties {
			result = result && suite.Contains(s.MetadataSchema().Type.Properties, property)
		}
		return result
	})
}

func (suite *EntrySchemaTestSuite) assertKeepResult(result map[string]bool, expected bool, values ...string) {
	for _, v := range values {
		msg := fmt.Sprintf("Expected result[%v] == %v", v, expected)
		if expected {
			suite.True(result[v], msg)
		} else {
			suite.False(result[v], msg)
		}
	}
}
