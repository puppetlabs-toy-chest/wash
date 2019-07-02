package types

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"strings"
	"testing"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

type EntrySchemaTestSuite struct {
	suite.Suite
}

func (suite *EntrySchemaTestSuite) TestNewEntrySchema() {
	s := suite.readFixture("complex")
	expected := map[string][]string{
		"A": []string{"B", "C"},
		"B": []string{"D", "E"},
		"C": []string{"C", "F"},
		"D": []string{},
		"E": []string{"A", "C"},
		"F": []string{},
	}
	suite.Equal(expected, s.toMap())

	// Ensure that duplicates are properly handled
	ABEC := suite.findNestedChild(s, "B", "E", "C")
	AC := suite.findNestedChild(s, "C")
	suite.True(ABEC == AC, "ABEC != AC")
	A := suite.findNestedChild(s)
	ABEA := suite.findNestedChild(A, "B", "E", "A")
	suite.True(A == ABEA, "A != ABEA")
}

func (suite *EntrySchemaTestSuite) TestNewEntrySchema_MetadataSchemaPValue_DefaultsToMetaAttributeSchema() {
	s := suite.readFixture("metadata")
	suite.Equal(s.MetadataSchemaPValue.Type.Type, "object")
	suite.Contains(s.MetadataSchemaPValue.Type.Properties, "foo")
}

func (suite *EntrySchemaTestSuite) TestNewEntrySchema_MetadataSchemaPValue_FullmetaSet_NilMetadataSchema_FallsbackToMetaAttributeSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata", opts)
	suite.Equal(s.MetadataSchemaPValue.Type.Type, "object")
	suite.Contains(s.MetadataSchemaPValue.Type.Properties, "foo")
}
func (suite *EntrySchemaTestSuite) TestNewEntrySchema_MetadataSchemaPValue_FullmetaSet_SetToMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata", opts)
	s = suite.findNestedChild(s, "B")
	suite.Equal(s.MetadataSchemaPValue.Type.Type, "array")
}

func (suite *EntrySchemaTestSuite) TestPrune_CanPruneRoot() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP()
	s = Prune(s, p)
	suite.Nil(s)
}

func (suite *EntrySchemaTestSuite) TestPrune_Tree() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP("D")
	s = Prune(s, p)
	expected := map[string][]string{
		"A": []string{"B"},
		"B": []string{"D"},
		"D": []string{},
	}
	suite.Equal(expected, s.toMap())
}

func (suite *EntrySchemaTestSuite) TestPrune_Complex() {
	s := suite.readFixture("complex")
	p := suite.makeSchemaP("D")
	s = Prune(s, p)
	expected := map[string][]string{
		"A": []string{"B"},
		"B": []string{"D", "E"},
		"D": []string{},
		"E": []string{"A"},
	}
	suite.Equal(expected, s.toMap())
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
	s = Prune(s, p)
	expected := map[string][]string{
		"aws.Root":            []string{"aws.profile"},
		"aws.profile":         []string{"aws.resourcesDir"},
		"aws.resourcesDir":    []string{"aws.ec2Dir"},
		"aws.ec2Dir":          []string{"aws.ec2InstancesDir"},
		"aws.ec2InstancesDir": []string{"aws.ec2Instance"},
		"aws.ec2Instance":     []string{},
	}
	suite.Equal(expected, s.toMap())
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Tree() {
	s := suite.readFixture("tree")

	p := suite.makeSchemaP("A", "B", "C", "D", "E", "F", "G")
	result := evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "C", "D", "E", "F", "G")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, false, "A", "B", "C", "D", "E", "F", "G")

	p = suite.makeSchemaP("D")
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "D")
	suite.assertKeepResult(result, false, "E", "C", "F", "G")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Triangle() {
	s := suite.readFixture("triangle")

	p := suite.makeSchemaP("A", "B", "C")
	result := evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "C")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, false, "A", "B", "C")

	p = suite.makeSchemaP("B")
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "C")
}

func (suite *EntrySchemaTestSuite) TestEvaluateNodesToKeep_Complex() {
	s := suite.readFixture("complex")

	p := suite.makeSchemaP("A", "B", "C", "D", "E", "F")
	result := evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "C", "D", "E", "F")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, false, "A", "B", "C", "D", "E", "F")

	p = suite.makeSchemaP("F")
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "C", "E", "F")
	suite.assertKeepResult(result, false, "D")

	p = suite.makeSchemaP("D")
	result = evaluateNodesToKeep(s, p)
	suite.assertKeepResult(result, true, "A", "B", "D", "E")
	suite.assertKeepResult(result, false, "C", "F")
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
		child = child.Children[segment]
		if child == nil {
			suite.T().Fatal(fmt.Sprintf("Child %v does not exist", strings.Join(visitedSegments, "/")))
		}
	}
	return child
}

func (suite *EntrySchemaTestSuite) readFixture(name string, options ...Options) *EntrySchema {
	filePath := path.Join("testdata", name+".json")
	rawSchema, err := ioutil.ReadFile(filePath)
	if err != nil {
		suite.T().Fatal(fmt.Sprintf("Failed to read %v", filePath))
	}
	var s *apitypes.EntrySchema
	if err := json.Unmarshal(rawSchema, &s); err != nil {
		suite.T().Fatal(fmt.Sprintf("Failed to unmarshal %v: %v", filePath, err))
	}
	opts := NewOptions()
	if len(options) > 0 {
		opts = options[0]
	}
	return NewEntrySchema(s, opts)
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
