package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/puppetlabs/wash/plugin/internal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockExternalPluginScript struct {
	mock.Mock
	path string
}

func (m *mockExternalPluginScript) Path() string {
	return m.path
}

func (m *mockExternalPluginScript) InvokeAndWait(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) (invocation, error) {
	retValues := m.Called(ctx, method, entry, args)
	return retValues.Get(0).(invocation), retValues.Error(1)
}

func (m *mockExternalPluginScript) NewInvocation(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) invocation {
	// A stub's still necessary to satisfy the externalPluginScript
	// interface
	panic("mockExternalPluginScript#NewInvocation called by tests")
}

// We make ctx an interface{} so that this method could
// be used when the caller generates a context using e.g.
// context.Background()
func (m *mockExternalPluginScript) OnInvokeAndWait(
	ctx interface{},
	method string,
	entry *externalPluginEntry,
	args ...string,
) *mock.Call {
	return m.On("InvokeAndWait", ctx, method, entry, args)
}

type ExternalPluginEntryTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryRequiredFields() {
	decodedEntry := decodedExternalPluginEntry{}

	_, err := decodedEntry.toExternalPluginEntry(false, false)
	suite.Regexp("name", err)
	decodedEntry.Name = "decodedEntry"

	_, err = decodedEntry.toExternalPluginEntry(false, false)
	suite.Regexp("methods", err)
	decodedEntry.Methods = []interface{}{"list"}

	_, err = decodedEntry.toExternalPluginEntry(false, false)
	suite.Regexp("type.*ID", err)
	decodedEntry.TypeID = "foo"

	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Equal(1, len(entry.methods))
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"])
		suite.Equal("foo", entry.typeID)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryExtraFields() {
	decodedEntry := decodedExternalPluginEntry{
		TypeID:  "foo",
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "stream"},
	}

	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"])
		suite.Contains(entry.methods, "stream")
		suite.Nil(entry.methods["stream"])
		suite.False(entry.isPrefetched())

		methods := entry.supportedMethods()
		suite.Equal(2, len(methods))
		suite.Contains(methods, "list")
		suite.Contains(methods, "stream")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithMethodResults() {
	childEntry := map[string]interface{}{"name": "foo", "methods": []string{"read"}}
	decodedEntry := decodedExternalPluginEntry{
		TypeID:  "foo",
		Name:    "decodedEntry",
		Methods: []interface{}{[]interface{}{"list", []interface{}{childEntry}}, "read"},
	}

	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.NotNil(entry.methods["list"])
		suite.Contains(entry.methods, "read")
		suite.Nil(entry.methods["read"])
		suite.True(entry.isPrefetched())
	}
}

func newMockDecodedEntry(name string) decodedExternalPluginEntry {
	return decodedExternalPluginEntry{
		TypeID:  "foo",
		Name:    name,
		Methods: []interface{}{"list"},
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithState() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.State = "some state"
	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.State, entry.state)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithCacheTTLs() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.CacheTTLs = decodedCacheTTLs{List: 1}
	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		expectedTTLs := NewEntry("foo").ttl
		expectedTTLs[ListOp] = decodedEntry.CacheTTLs.List * time.Second
		suite.Equal(expectedTTLs, entry.EntryBase.ttl)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSlashReplacer() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.SlashReplacer = "a string"
	suite.Panics(
		func() { _, _ = decodedEntry.toExternalPluginEntry(false, false) },
		"e.SlashReplacer: received string a string instead of a character",
	)
	decodedEntry.SlashReplacer = ":"
	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.Equal(':', entry.slashReplacer())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithAttributes() {
	decodedEntry := newMockDecodedEntry("name")
	t := time.Now()
	decodedEntry.Attributes.SetCtime(t)
	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		expectedAttr := EntryAttributes{}
		expectedAttr.SetCtime(t)
		suite.Equal(expectedAttr, entry.attr)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_DoesNotImplementSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list"},
		TypeID:  "foo",
	}
	entry, err := decodedEntry.toExternalPluginEntry(false, false)
	if suite.NoError(err) {
		suite.False(entry.schemaKnown)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(false, false)
	suite.Regexp("decodedEntry.*foo.*implements.*schema.*root", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_DoesNotImplementSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list"},
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(true, false)
	suite.Regexp("decodedEntry.*foo.*must.*implement.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_PrefetchesSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name: "decodedEntry",
		Methods: []interface{}{
			"list",
			[]interface{}{"schema", "schema_result"},
		},
		TypeID: "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(true, false)
	suite.Regexp("decodedEntry.*foo.*plugin.*roots.*support.*prefetching", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
		TypeID:  "foo",
	}
	entry, err := decodedEntry.toExternalPluginEntry(true, false)
	if suite.NoError(err) {
		suite.True(entry.schemaKnown)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSetCacheTTLs() {
	decodedTTLs := decodedCacheTTLs{
		List:     10,
		Read:     15,
		Metadata: 20,
	}

	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
	}
	entry.setCacheTTLs(decodedTTLs)

	suite.Equal(decodedTTLs.List*time.Second, entry.getTTLOf(ListOp))
	suite.Equal(decodedTTLs.Read*time.Second, entry.getTTLOf(OpenOp))
	suite.Equal(decodedTTLs.Metadata*time.Second, entry.getTTLOf(MetadataOp))
}

func mockInvocation(stdout []byte) invocation {
	return invocation{command: internal.NewCommand(context.Background(), ""), stdout: *bytes.NewBuffer(stdout)}
}

// TODO: Add tests for Schema, including when schemaGraph is provided (prefetched)
// and when it is not provided

func (suite *ExternalPluginEntryTestSuite) TestSchema_DoesNotImplementSchema_ReturnsNil() {
	entry := &externalPluginEntry{}
	s, err := entry.schema()
	if suite.NoError(err) {
		suite.Nil(s)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_PanicsIfNoSchemaGraphWasProvided() {
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
	}
	entry.SetTestID("/foo")

	suite.Panics(
		func() { _, _ = entry.schema() },
		"e.Schema(): entry schemas were prefetched, but no schema graph was provided for /foo (type ID fooTypeID)",
	)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsTheSchemaGraph() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/foo")
	graph := linkedhashmap.New()
	graph.Put(
		entry.typeID,
		entrySchema{
			Actions: []string{"schema"},
		},
	)
	entry.schemaGraphs[entry.typeID] = graph

	s, err := entry.schema()
	if suite.NoError(err) {
		suite.Equal(entry.schemaGraphs[entry.typeID], s.graph)
		// Make sure that Wash did not shell out to the plugin script
		mockScript.AssertNotCalled(suite.T(), "InvokeAndWait")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsErrorIfSchemaAndInstanceMethodsDontMatch() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
			"read":   nil,
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/foo")
	graph := linkedhashmap.New()
	graph.Put(
		entry.typeID,
		entrySchema{
			Actions: []string{"list", "exec"},
		},
	)
	entry.schemaGraphs[entry.typeID] = graph

	_, err := entry.schema()
	suite.Regexp("schema.*methods.*exec.*list.*instance.*methods.*read.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfInvocationFails() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
		},
		script: mockScript,
	}
	entry.SetTestID("/foo")

	invokeErr := fmt.Errorf("invocation failed")
	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte{}), invokeErr).Once()
	_, err := entry.schema()
	suite.Regexp("foo.*fooTypeID.*invocation.*failed", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfUnmarshallingFails() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
		},
		script: mockScript,
	}
	entry.SetTestID("/foo")

	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte("\"foo\"")), nil).Once()
	_, err := entry.schema()
	suite.Regexp("/foo.*fooTypeID", err)
	suite.Regexp(regexp.QuoteMeta(schemaFormat), err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_SuccessfulInvocation_ReturnsSchema() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		typeID:    "fooTypeID",
		methods: map[string]interface{}{
			"schema": nil,
		},
		script: mockScript,
	}
	entry.SetTestID("/foo")

	stdout := `
{
	"fooTypeID": {
		"label": "fooEntry",
		"methods": ["list"],
		"children": ["barTypeID"],
		"singleton": true,
		"meta_attribute_schema": {
			"type": "object"
		},
		"metadata_schema": null
	},
	"barTypeID": {
		"label": "barEntry",
		"methods": ["list"],
		"children": ["barTypeID"],
		"singleton": false,
		"meta_attribute_schema": {
			"type": "object",
			"properties": {
				"foo": {
					"type": "string"
				}
			}
		},
		"metadata_schema": null
	}
}
`
	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte(stdout)), nil).Once()
	schema, err := entry.schema()
	if suite.NoError(err) && suite.NotNil(schema) {
		schemaJSON, err := json.Marshal(schema)
		if suite.NoError(err) {
			stdout = strings.ReplaceAll(stdout, "methods", "actions")
			suite.JSONEq(stdout, string(schemaJSON))
		}
	}
}

func (suite *ExternalPluginEntryTestSuite) TestList() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
		schemaGraphs: map[string]*linkedhashmap.Map{
			"foo": linkedhashmap.New(),
		},
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then List returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.List(ctx)
	suite.EqualError(mockErr, err.Error())

	// Test that List returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	_, err = entry.List(ctx)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that List properly decodes the entries from stdout
	stdout := "[" +
		"{\"name\":\"foo\",\"methods\":[\"list\"],\"type_id\":\"foo\"}" +
		"]"
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		entryBase := NewEntry("foo")
		expectedEntries := []Entry{
			&externalPluginEntry{
				EntryBase:    entryBase,
				methods:      map[string]interface{}{"list": nil},
				script:       entry.script,
				typeID:       "foo",
				schemaGraphs: entry.schemaGraphs,
			},
		}

		suite.Equal(expectedEntries, entries)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestOpen() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "read", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Open returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.Open(ctx)
	suite.EqualError(mockErr, err.Error())

	// Test that Open wraps all of stdout into a SizedReader
	stdout := "foo"
	mockInvokeAndWait([]byte(stdout), nil)
	rdr, err := entry.Open(ctx)
	if suite.NoError(err) {
		expectedRdr := bytes.NewReader([]byte(stdout))
		suite.Equal(expectedRdr, rdr)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestListOpenWithMethodResults() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that List is invoked when
	stdoutFn := func(content string) []byte {
		return []byte(`
[{"name": "foo", "type_id": "bar", "methods": [
	["list", [
		{"name": "bar", "type_id": "baz", "methods": [["read", "` + content + `"]]}
	]]
]}]`)
	}
	someContent := "some content"
	mockInvokeAndWait([]byte(stdoutFn(someContent)), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries))
		if suite.Equal([]string{"list"}, SupportedActionsOf(entries[0])) {
			children, err := entries[0].(Parent).List(ctx)
			if suite.NoError(err) {
				suite.Equal(1, len(children))
				attr := Attributes(children[0])
				if suite.True(attr.HasSize()) {
					suite.Equal(uint64(len(someContent)), attr.Size())
				}

				if suite.Equal([]string{"read"}, SupportedActionsOf(children[0])) {
					rdr, err := children[0].(Readable).Open(ctx)
					suite.NoError(err)
					buf := make([]byte, rdr.Size())
					_, err = rdr.ReadAt(buf, 0)
					suite.NoError(err)
					suite.Equal(someContent, string(buf))
				}
			}
		}
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeWithErrors() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that List is invoked when
	stdout := `[{"name": "foo", "type_id": "bar", "methods": [
								["list", {"name": "bar"}],
								["read", [1, 2]]
							]}]`
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries))
		supported := SupportedActionsOf(entries[0])
		suite.Contains(supported, "list")
		suite.Contains(supported, "read")

		_, err = entries[0].(Parent).List(ctx)
		suite.EqualError(err, `implementation of list must conform to `+
			`[{"name":"entry1","methods":["list"],"type_id":"type1"},{"name":"entry2","methods":["list"],"type_id":"type2"}], not map[name:bar]`)

		_, err = entries[0].(Readable).Open(ctx)
		suite.EqualError(err, "Read method must provide a string, not [1 2]")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestMetadata_NotImplemented() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	expectedMeta := JSONObject{"foo": "bar"}
	entry.Attributes().SetMeta(expectedMeta)

	// If metadata is not implemented, then e.Metadata should return
	// EntryBase#Metadata, which returns the meta attribute.
	meta, err := entry.Metadata(context.Background())
	if suite.NoError(err) {
		suite.Equal(expectedMeta, meta)
	}
	// Make sure that Wash did not shell out to the plugin script
	mockScript.AssertNotCalled(suite.T(), "InvokeAndWait")
}

func (suite *ExternalPluginEntryTestSuite) TestMetadata_Implemented() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		methods:   map[string]interface{}{"metadata": nil},
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "metadata", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Metadata returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.Metadata(ctx)
	suite.EqualError(mockErr, err.Error())

	// Test that Metadata returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	_, err = entry.Metadata(ctx)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Metadata properly decodes the entries from stdout
	stdout := "{\"key\":\"value\"}"
	mockInvokeAndWait([]byte(stdout), nil)
	metadata, err := entry.Metadata(ctx)
	if suite.NoError(err) {
		expectedMetadata := JSONObject{"key": "value"}
		suite.Equal(expectedMetadata, metadata)
	}
}

// TODO: Add tests for stdoutStreamer, Stream and Exec
// once the API for Stream and Exec's at a more stable
// state.

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotAJSONObject() {
	entry := &externalPluginEntry{}
	_, err := entry.unmarshalSchemaGraph([]byte("[]"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfAnEmptyJSONObject() {
	entry := &externalPluginEntry{}
	_, err := entry.unmarshalSchemaGraph([]byte("{}"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfTypeIDNotPresent() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}
	stdout := []byte(`
{
	"bar": "baz"
}`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("foo.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnMalformedSchema() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	// Error should indicate that foo's schema is not a JSON object.
	stdout := []byte(`
{
	"foo": "fooSchema",
	"bar": {}
}`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("object.*foo.*fooSchema", err)

	// Error should indicate that "foo"'s label is malformed.
	stdout = []byte(`
{
	"foo": {
		"label": 5
	},
	"bar": {}
}`)
	_, err = entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("number", err)

	// Error should indicate that "bar"'s children are malformed.
	stdout = []byte(`
{
	"foo": {
		"label": "foo_label",
		"methods": ["read"]
	},
	"bar": {
		"children": true
	}
}`)
	_, err = entry.unmarshalSchemaGraph(stdout)
	suite.Regexp(`\[\]string`, err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfLabelNotProvided() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("label.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMethodsNotProvided() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel"
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("methods.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotParentAndChildrenProvided() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"],
		"children": ["barTypeID"]
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("entry.*children.*not.*parent", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfParentAndChildrenNotProvided() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["list"],
		"children": []
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("parent.*entries.*children", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMissingChildSchema() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["list"],
		"children": ["bar"]
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("bar.*schema.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfSchemaIncludesDanglingTypeIDs() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"]
	},
	"bar": {
		"label": "barLabel",
		"methods": ["read"]
	},
	"baz": {
		"label": "bazLabel",
		"methods": ["read"]
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	// Need to do several regexp checks here b/c this error
	// message is generated by iterating over a map's keys.
	// Map keys are iterated in random order.
	suite.Regexp("bar", err)
	suite.Regexp("baz", err)
	suite.Regexp("associated", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidMetaAttributeSchema() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"],
		"meta_attribute_schema": {
			"type": "array"
		}
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("invalid.*meta.*attribute.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidMetadataSchema() {
	entry := &externalPluginEntry{
		typeID: "foo",
	}

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"],
		"metadata_schema": {
			"type": "array"
		}
	}
}
`)
	_, err := entry.unmarshalSchemaGraph(stdout)
	suite.Regexp("invalid.*metadata.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ValidInput() {
	// This test tests unmarshalSchemaGraph by ensuring that the returned graph
	// can marshal back into its original form (with the "methods" key replaced
	// by the "actions" key)
	stdout := readSchemaFixture(suite.Suite, "externalPluginSchema")

	entry := &externalPluginEntry{
		typeID: "aws.Root",
	}
	actualGraph, err := entry.unmarshalSchemaGraph(stdout)
	if suite.NoError(err) {
		// Check that the first key is aws.Root
		it := actualGraph.Iterator()
		it.First()
		suite.Equal(it.Key(), entry.typeID)

		// Now check that all of stdout was successfully unmarshalled.
		stdout = bytes.ReplaceAll(stdout, []byte("methods"), []byte("actions"))
		actualGraphJSON, err := actualGraph.ToJSON()
		if suite.NoError(err) {
			suite.JSONEq(string(stdout), string(actualGraphJSON))
		}
	}
}

func TestExternalPluginEntry(t *testing.T) {
	suite.Run(t, new(ExternalPluginEntryTestSuite))
}

// This helper's also used by the external plugin root tests
func readSchemaFixture(suite suite.Suite, name string) []byte {
	filePath := path.Join("testdata", name+".json")
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		suite.FailNow(fmt.Sprintf("Failed to read %v", filePath))
		return nil
	}
	return bytes
}
