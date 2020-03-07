package rql

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

// NOTE: Prune munges the schema's path to store its kind.
// Thus, s.ToMap() will return a map of <kind> => children
// instead of <path> => children. Remember that an entry's
// path is of the form <root_label>/<parentOne_label>/.../<label>.
// Thus, the root's kind is empty ("") while a non-root entry's
// kind is <parentOne_label>/.../<label>.

type PruneTestSuite struct {
	suite.Suite
}

func (suite *PruneTestSuite) TestPrune_CanPruneRoot() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP()
	s = prune(s, p, NewOptions())
	suite.Nil(s)
}

func (suite *PruneTestSuite) TestPrune_Tree() {
	s := suite.readFixture("tree")
	p := suite.makeSchemaP("B/D")
	s = prune(s, p, NewOptions())
	expected := map[string][]string{
		"":    []string{"B"},
		"B":   []string{"B/D"},
		"B/D": []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *PruneTestSuite) TestPrune_Complex() {
	s := suite.readFixture("complex")
	p := suite.makeSchemaP("B/D")
	s = prune(s, p, NewOptions())
	expected := map[string][]string{
		"":    []string{"B"},
		"B":   []string{"B/D", "B/E"},
		"B/D": []string{},
		"B/E": []string{""},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *PruneTestSuite) TestPrune_RealWorld_AWS() {
	s := suite.readFixture("real_world_aws")
	p := &mockQuery{
		EntrySchemaP: func(s *EntrySchema) bool {
			for _, action := range s.Actions() {
				if action == "exec" {
					return true
				}
			}
			return false
		},
	}
	s = prune(s, p, NewOptions())
	expected := map[string][]string{
		"":                                []string{"profile"},
		"profile":                         []string{"profile/resources"},
		"profile/resources":               []string{"profile/resources/ec2"},
		"profile/resources/ec2":           []string{"profile/resources/ec2/instances"},
		"profile/resources/ec2/instances": []string{"profile/resources/ec2/instances/instance"},
		"profile/resources/ec2/instances/instance": []string{},
	}
	suite.Equal(expected, s.ToMap())
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_Tree() {
	s := suite.readFixture("tree")

	p := suite.makeSchemaP("", "B", "C", "B/D", "B/E", "C/F", "C/G")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "C", "B/D", "B/E", "C/F", "C/G")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "", "B", "C", "B/D", "B/E", "C/F", "C/G")

	p = suite.makeSchemaP("B/D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "B/D")
	suite.assertKeepResult(result, false, "B/E", "C", "C/F", "C/G")
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_Triangle() {
	s := suite.readFixture("triangle")

	p := suite.makeSchemaP("", "B", "B/C")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "B/C")

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, false, "", "B", "B/C")

	p = suite.makeSchemaP("B")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "B/C")
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_Complex() {
	s := suite.readFixture("complex")

	p := suite.makeSchemaP(
		"",
		"B",
		"C",
		"B/D",
		"B/E",
		"C/F",
		"B/E/C",
		"B/E/C/F",
	)
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(
		result,
		true,
		"",
		"B",
		"C",
		"B/D",
		"B/E",
		"C/F",
		"B/E/C",
		"B/E/C/F",
	)

	p = suite.makeSchemaP()
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(
		result,
		false,
		"",
		"B",
		"C",
		"B/D",
		"B/E",
		"C/F",
		"B/E/C",
		"B/E/C/F",
	)

	p = suite.makeSchemaP("B/E/C/F")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "B/E", "B/E/C", "B/E/C/F")
	suite.assertKeepResult(result, false, "B/D", "C", "C/F")

	p = suite.makeSchemaP("B/D")
	result = evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "", "B", "B/D", "B/E")
	suite.assertKeepResult(result, false, "B/E/C", "B/E/C/F", "C", "C/F")
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_MetadataSchema_DefaultsToPartialMetadataSchema() {
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("", "object", "foo")
	result := evaluateNodesToKeep(s, p, NewOptions())
	suite.assertKeepResult(result, true, "")
	suite.assertKeepResult(result, false, "B", "C")
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_NilMetadataSchema_FallsbackToPartialMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("", "object", "foo")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "")
	suite.assertKeepResult(result, false, "B", "C")
}

func (suite *PruneTestSuite) TestEvaluateNodesToKeep_MetadataSchema_FullmetaSet_SetToMetadataSchema() {
	opts := NewOptions()
	opts.Fullmeta = true
	s := suite.readFixture("metadata")
	p := suite.makeMetadataSchemaP("B", "array")
	result := evaluateNodesToKeep(s, p, opts)
	suite.assertKeepResult(result, true, "", "B")
	suite.assertKeepResult(result, false, "C")
}

func TestEntrySchema(t *testing.T) {
	suite.Run(t, new(PruneTestSuite))
}

func (suite *PruneTestSuite) readFixture(name string) *EntrySchema {
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

func (suite *PruneTestSuite) makeSchemaP(trueValues ...string) EntrySchemaPredicate {
	return &mockQuery{
		EntrySchemaP: func(s *EntrySchema) bool {
			for _, path := range trueValues {
				if s.Path() == path {
					return true
				}
			}
			return false
		},
	}
}

func (suite *PruneTestSuite) makeMetadataSchemaP(trueValue string, expectedType string, expectedProperties ...string) EntrySchemaPredicate {
	return &mockQuery{
		EntrySchemaP: func(s *EntrySchema) bool {
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
		},
	}
}

func (suite *PruneTestSuite) assertKeepResult(result map[string]bool, expected bool, values ...string) {
	for _, v := range values {
		msg := fmt.Sprintf("Expected result[%v] == %v", v, expected)
		if expected {
			suite.True(result[v], msg)
		} else {
			suite.False(result[v], msg)
		}
	}
}
