package plugin

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/emirpasic/gods/maps/linkedhashmap"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ExternalPluginRootTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginRootTestSuite) TestInit() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(
			mock.Anything,
			"init",
			nil,
			"{}",
		).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Init returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	err := root.Init(nil)
	suite.EqualError(mockErr, err.Error())

	// Test that Init returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	err = root.Init(nil)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Init properly decodes the root from stdout
	stdout := "{\"type_id\":\"foo_type\"}"
	mockInvokeAndWait([]byte(stdout), nil)
	err = root.Init(nil)
	if suite.NoError(err) {
		expectedRoot := &externalPluginRoot{
			externalPluginEntry: &externalPluginEntry{
				EntryBase: NewEntry("foo"),
				methods:   map[string]interface{}{"list": nil},
				script:    root.script,
				typeID:    "foo::foo_type",
			},
		}

		suite.Equal(expectedRoot, root)
	}
}

func (suite *ExternalPluginRootTestSuite) TestInitWithConfig() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockScript.OnInvokeAndWait(
		mock.Anything,
		"init",
		nil,
		`{"key":["value"]}`,
	).Return(mockInvocation([]byte("{}")), nil).Once()

	suite.NoError(root.Init(map[string]interface{}{"key": []string{"value"}}))
}

func (suite *ExternalPluginRootTestSuite) TestInitWithSchema_SetsSchemaKnownVariable() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockScript.OnInvokeAndWait(
		mock.Anything,
		"init",
		nil,
		"{}",
	).Return(mockInvocation([]byte("{\"type_id\":\"root\",\"methods\":[\"schema\",\"list\"]}")), nil).Once()

	suite.NoError(root.Init(nil))
	suite.True(root.schemaKnown)
}

func (suite *ExternalPluginRootTestSuite) TestInitWithSchema_PrefetchedSchema_ReturnsErrorIfUnmarshallingSchemaFails() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockScript.OnInvokeAndWait(
		mock.Anything,
		"init",
		nil,
		"{}",
	).Return(mockInvocation([]byte("{\"type_id\":\"root\",\"methods\":[[\"schema\", \"foo\"],\"list\"]}")), nil).Once()

	err := root.Init(nil)
	suite.Regexp("decode.*schema", err)
	suite.Regexp("object", err)
	suite.Regexp("foo", err)
	suite.Regexp(regexp.QuoteMeta(schemaFormat), err)
}

func (suite *ExternalPluginRootTestSuite) TestInitWithSchema_PrefetchedSchema_PartitionsSchemaGraph() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("fooPlugin"),
		script:    mockScript,
	}}

	// Set-up the mocks
	schemaJSON := readSchemaFixture(suite.Suite, "externalPluginSchema_SchemaGraph")
	var unmarshalledSchemaJSON map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &unmarshalledSchemaJSON); err != nil {
		suite.FailNowf("received unexpected error while unmarshalling the schema: %v", err.Error())
	}
	stdoutMap := map[string]interface{}{
		"type_id": "aws.Root",
		"methods": []interface{}{
			"list",
			[]interface{}{
				"schema",
				unmarshalledSchemaJSON,
			},
		},
	}
	stdout, err := json.Marshal(stdoutMap)
	if err != nil {
		suite.FailNowf("received unexpected error while marshalling the input: %v", err.Error())
		return
	}
	mockScript.OnInvokeAndWait(
		mock.Anything,
		"init",
		nil,
		"{}",
	).Return(mockInvocation(stdout), nil).Once()

	// Perform the test
	expectedGraph := readSchemaFixture(suite.Suite, "externalPluginSchema_PartitionedSchemaGraph")
	if err := root.Init(nil); suite.NoError(err) && suite.NotNil(root.schemaGraphs) {
		// Ensure that the graph of root.schemaGraphs[type_id] has "type_id" as its
		// first item. We pick an arbitrary type ID here
		typeID := namespace(root.name(), "aws.profile")
		graph := root.schemaGraphs[typeID]
		if suite.NotNil(graph) {
			it := graph.Iterator()
			it.First()
			suite.Equal(typeID, it.Key())
		}

		// Now ensure that the rest of root.schemaGraphs is what we expect. We already
		// checked that "type_id" is the first item, so here it is enough to check that
		// the data matches.
		wrappedSchemaGraphs := make(map[string]orderedMap)
		for typeID, graph := range root.schemaGraphs {
			wrappedSchemaGraphs[typeID] = orderedMap{graph}
		}
		marshalledGraph, err := json.Marshal(wrappedSchemaGraphs)
		if err != nil {
			suite.FailNowf("received unexpected error while marshalling the returned schema graphs: %v", err.Error())
		}
		suite.JSONEq(string(expectedGraph), string(marshalledGraph))
	}
}

func TestExternalPluginRoot(t *testing.T) {
	suite.Run(t, new(ExternalPluginRootTestSuite))
}

// This wrapped type's here because linkedhashmap doesn't implement the
// json.Marshaler/json.Unmarshaler interfaces, which makes marshalling/unmarshalling
// it very annoying, especially when testing.
//
// We don't mix this type in with the production code because it will (hopefully)
// be removed once https://github.com/emirpasic/gods/issues/116 is fixed
type orderedMap struct {
	*linkedhashmap.Map
}

func (mp orderedMap) MarshalJSON() ([]byte, error) {
	return mp.ToJSON()
}
