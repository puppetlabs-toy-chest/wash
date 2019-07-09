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
	p := suite.makeSchemaP("D")
	s = Prune(s, p, NewOptions())
	expected := map[string][]string{
		"A": []string{"B"},
		"B": []string{"D"},
		"D": []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *EntrySchemaTestSuite) TestPrune_Complex() {
	s := suite.readFixture("complex")
	p := suite.makeSchemaP("D")
	s = Prune(s, p, NewOptions())
	expected := map[string][]string{
		"A": []string{"B"},
		"B": []string{"D", "E"},
		"D": []string{},
		"E": []string{"A"},
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
		"aws.Root":            []string{"aws.profile"},
		"aws.profile":         []string{"aws.resourcesDir"},
		"aws.resourcesDir":    []string{"aws.ec2Dir"},
		"aws.ec2Dir":          []string{"aws.ec2InstancesDir"},
		"aws.ec2InstancesDir": []string{"aws.ec2Instance"},
		"aws.ec2Instance":     []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Tree() {
	s := suite.readFixture("tree")

	p := suite.makeSchemaP("A", "B", "C", "D", "E", "F", "G")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "C", "D", "E", "F", "G")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "A", "B", "C", "D", "E", "F", "G")

	p = suite.makeSchemaP("D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "D")
	suite.assertKeepResult(result, false, "E", "C", "F", "G")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Triangle() {
	s := suite.readFixture("triangle")

	p := suite.makeSchemaP("A", "B", "C")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "C")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "A", "B", "C")

	p = suite.makeSchemaP("B")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Complex() {
	s := suite.readFixture("complex")

	p := suite.makeSchemaP("A", "B", "C", "D", "E", "F")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "C", "D", "E", "F")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "A", "B", "C", "D", "E", "F")

	p = suite.makeSchemaP("F")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "C", "E", "F")
	suite.assertKeepResult(result, false, "D")

	p = suite.makeSchemaP("D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A", "B", "D", "E")
	suite.assertKeepResult(result, false, "C", "F")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_DefaultsToMetaAttributeSchema() {
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("A", "object", "foo")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "A")
	suite.assertKeepResult(result, false, "B", "C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_NilMetadataSchema_FallsbackToMetaAttributeSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("A", "object", "foo")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "A")
	suite.assertKeepResult(result, false, "B", "C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_SetToMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("B", "array")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "A", "B")
	suite.assertKeepResult(result, false, "C")
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
		for _, typeID := range trueValues {
			if s.TypeID() == typeID {
				return true
			}
		}
		return false
	})
}

func (suite *EntrySchemaTestSuite) makeMetadataSchemaP(trueValue string, expectedType string, expectedProperties ...string) EntrySchemaPredicate {
	return ToEntrySchemaP(func(s *EntrySchema) bool {
		if s.TypeID() != trueValue {
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
