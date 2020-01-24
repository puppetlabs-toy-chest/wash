package external

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
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockPluginScript struct {
	mock.Mock
	path string
}

func (m *mockPluginScript) Path() string {
	return m.path
}

func (m *mockPluginScript) InvokeAndWait(
	ctx context.Context,
	method string,
	entry *pluginEntry,
	args ...string,
) (invocation, error) {
	retValues := m.Called(ctx, method, entry, args)
	return retValues.Get(0).(invocation), retValues.Error(1)
}

func (m *mockPluginScript) NewInvocation(
	ctx context.Context,
	method string,
	entry *pluginEntry,
	args ...string,
) invocation {
	retValues := m.Called(ctx, method, entry, args)
	return retValues.Get(0).(invocation)
}

// We make ctx an interface{} so that this method could
// be used when the caller generates a context using e.g.
// context.Background()
func (m *mockPluginScript) OnInvokeAndWait(
	ctx interface{},
	method string,
	entry *pluginEntry,
	args ...string,
) *mock.Call {
	return m.On("InvokeAndWait", ctx, method, entry, args)
}

type ExternalPluginEntryTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryRequiredFields() {
	decodedEntry := decodedExternalPluginEntry{}

	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("name", err)
	decodedEntry.Name = "decodedEntry"

	_, err = decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("methods", err)
	decodedEntry.Methods = rawMethods(`"list"`)

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, plugin.Name(entry))
		suite.Equal(1, len(entry.methods))
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"].tupleValue)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryExtraFields() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `"stream"`),
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.Name())
		suite.Contains(entry.methods, "list")
		suite.Equal(plugin.DefaultSignature, entry.methods["list"].signature)
		suite.Nil(entry.methods["list"].tupleValue)
		suite.Contains(entry.methods, "stream")
		suite.Equal(plugin.DefaultSignature, entry.methods["stream"].signature)
		suite.Nil(entry.methods["stream"].tupleValue)
		suite.False(plugin.IsPrefetched(entry))

		methods := entry.supportedMethods()
		suite.Equal(2, len(methods))
		suite.Contains(methods, "list")
		suite.Contains(methods, "stream")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntry_SupportsEmptyMethodsArray() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []json.RawMessage{},
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.Name())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithMethodResults() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`["list", [{"name": "foo", "methods": ["read"]}]]`, `"read"`),
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.Name())
		suite.Contains(entry.methods, "list")
		suite.Equal(plugin.DefaultSignature, entry.methods["list"].signature)
		suite.NotNil(entry.methods["list"].tupleValue)
		suite.Contains(entry.methods, "read")
		suite.Nil(entry.methods["read"].tupleValue)
		suite.True(plugin.IsPrefetched(entry))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryMethodTuple_Read() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`["read", true]`),
	}

	type testCase struct {
		data              interface{}
		expectedSignature plugin.MethodSignature
		expectedResult    interface{}
	}
	testCases := []testCase{
		testCase{true, plugin.BlockReadableSignature, nil},
		testCase{false, plugin.DefaultSignature, nil},
		testCase{"foo", plugin.DefaultSignature, []byte("foo")},
	}
	for _, testCase := range testCases {
		data, err := json.Marshal(testCase.data)
		if err != nil {
			panic(err)
		}
		decodedEntry.Methods = rawMethods(`["read", ` + string(data) + `]`)
		entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
		if suite.NoError(err) {
			suite.NotNil(entry.methods["read"])
			suite.Equal(testCase.expectedSignature, entry.methods["read"].signature)
			suite.Equal(testCase.expectedResult, entry.methods["read"].tupleValue)
		}
	}
}

func newMockDecodedEntry(name string) decodedExternalPluginEntry {
	return decodedExternalPluginEntry{
		Name:    name,
		Methods: rawMethods(`"list"`),
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithState() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.State = "some state"
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.State, entry.state)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithCacheTTLs() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.CacheTTLs = decodedCacheTTLs{List: 1}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.CacheTTLs.List*time.Second, entry.TTLOf(plugin.ListOp))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSlashReplacer() {
	decodedEntry := newMockDecodedEntry("name/")
	decodedEntry.SlashReplacer = "a string"
	suite.Panics(
		func() { _, _ = decodedEntry.toExternalPluginEntry(context.Background(), false, false) },
		"e.SlashReplacer: received string a string instead of a character",
	)
	decodedEntry.SlashReplacer = ":"
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal("name:", plugin.CName(entry))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithAttributes() {
	decodedEntry := newMockDecodedEntry("name")
	t := time.Now()
	decodedEntry.Attributes.SetCtime(t)
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		expectedAttr := plugin.EntryAttributes{}
		expectedAttr.SetCtime(t)
		suite.Equal(expectedAttr, plugin.Attributes(entry))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithPartialMetadata() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.PartialMetadata = plugin.JSONObject{}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(plugin.JSONObject{}, plugin.PartialMetadata(entry))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_DoesNotImplementSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`),
	}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.False(entry.schemaKnown)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_ImplementsSchema_TypeIDNotIncluded() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `"schema"`),
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("decodedEntry.*implements.*schema.*no.*type.*ID", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `"schema"`),
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("decodedEntry.*foo.*implements.*schema.*root", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_DoesNotImplementSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`),
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*foo.*must.*implement.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_ImplementsSchema_TypeIDNotIncluded() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `"schema"`),
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*implements.*schema.*no.*type.*ID", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_PrefetchesSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `["schema", {"foo": {"label": "foo", "methods": []}}]`),
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*foo.*plugin.*roots.*support.*prefetching", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: rawMethods(`"list"`, `"schema"`),
		TypeID:  "foo",
	}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	if suite.NoError(err) {
		suite.True(entry.schemaKnown)
		suite.Equal("foo", entry.RawTypeID())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithInaccessibleReason() {
	decodedEntry := decodedExternalPluginEntry{
		Name:               "decodedEntry",
		Methods:            rawMethods(`"list"`, `"stream"`),
		InaccessibleReason: "permission denied",
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.True(entry.IsInaccessible())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSetCacheTTLs() {
	decodedTTLs := decodedCacheTTLs{
		List:     10,
		Read:     15,
		Metadata: 20,
	}

	entry := pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
	}
	entry.setCacheTTLs(decodedTTLs)

	suite.Equal(decodedTTLs.List*time.Second, entry.TTLOf(plugin.ListOp))
	suite.Equal(decodedTTLs.Read*time.Second, entry.TTLOf(plugin.ReadOp))
	suite.Equal(decodedTTLs.Metadata*time.Second, entry.TTLOf(plugin.MetadataOp))
}

func mockInvocation(stdout []byte) invocation {
	return &invocationImpl{Command: NewCommand(context.Background(), ""), stdout: *bytes.NewBuffer(stdout)}
}

// TODO: Add tests for Schema, including when schemaGraph is provided (prefetched)
// and when it is not provided

func (suite *ExternalPluginEntryTestSuite) TestSchema_DoesNotImplementSchema_ReturnsNil() {
	entry := &pluginEntry{}
	s, err := entry.SchemaGraph()
	if suite.NoError(err) {
		suite.Nil(s)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_PanicsIfNoSchemaGraphWasProvided() {
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
	}
	entry.SetTestID("/fooPlugin")

	suite.Panics(
		func() { _, _ = entry.SchemaGraph() },
		"e.Schema(): entry schemas were prefetched, but no schema graph was provided for /foo (type ID fooTypeID)",
	)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsTheSchema() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/fooPlugin")
	var schema plugin.EntrySchema
	schema.Actions = []string{"schema"}
	graph := linkedhashmap.New()
	graph.Put(
		plugin.TypeID(entry),
		schema,
	)
	entry.schemaGraphs[plugin.TypeID(entry)] = graph

	graph, err := entry.SchemaGraph()
	if suite.NoError(err) {
		suite.Equal(entry.schemaGraphs[plugin.TypeID(entry)], graph)
		// Make sure that Wash did not shell out to the plugin script
		mockScript.AssertNotCalled(suite.T(), "InvokeAndWait")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsErrorIfSchemaAndInstanceMethodsDontMatch() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
			"read":   methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/fooPlugin")
	var schema plugin.EntrySchema
	schema.Actions = []string{"list", "exec"}
	graph := linkedhashmap.New()
	graph.Put(
		plugin.TypeID(entry),
		schema,
	)
	entry.schemaGraphs[plugin.TypeID(entry)] = graph

	_, err := entry.SchemaGraph()
	suite.Regexp("schema.*methods.*exec.*list.*instance.*methods.*read.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfInvocationFails() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		script: mockScript,
	}
	entry.SetTestID("/fooPlugin")

	invokeErr := fmt.Errorf("invocation failed")
	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte{}), invokeErr).Once()
	_, err := entry.SchemaGraph()
	suite.Regexp("foo.*fooTypeID.*invocation.*failed", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfUnmarshallingFails() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		script: mockScript,
	}
	entry.SetTestID("/fooPlugin")

	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte("\"foo\"")), nil).Once()
	_, err := entry.SchemaGraph()
	suite.Regexp("/foo.*fooTypeID", err)
	suite.Regexp(regexp.QuoteMeta(schemaFormat), err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_SuccessfulInvocation_ReturnsSchema() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		rawTypeID: "baz.fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		script: mockScript,
	}
	entry.SetTestID("/fooPlugin")

	stdout := `
{
	"baz.fooTypeID": {
		"label": "fooEntry",
		"methods": ["list"],
		"children": ["baz.barTypeID"],
		"singleton": true,
		"partial_metadata_schema": {
			"type": "object"
		},
		"metadata_schema": null
	},
	"baz.barTypeID": {
		"label": "barEntry",
		"methods": ["list"],
		"children": ["baz.barTypeID"],
		"singleton": false,
		"partial_metadata_schema": {
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
	graph, err := entry.SchemaGraph()
	if suite.NoError(err) && suite.NotNil(graph) {
		schemaJSON, err := graph.ToJSON()
		if suite.NoError(err) {
			stdout = strings.ReplaceAll(stdout, "methods", "actions")
			stdout = strings.ReplaceAll(stdout, "baz.", "fooPlugin::baz.")
			suite.JSONEq(stdout, string(schemaJSON))
		}
	}
}

func (suite *ExternalPluginEntryTestSuite) TestList() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
		schemaGraphs: map[string]*linkedhashmap.Map{
			"foo": linkedhashmap.New(),
		},
		rawTypeID: "foo_type",
	}
	entry.SetTestID("/fooPlugin")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then List returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.List(ctx)
	suite.EqualError(err, mockErr.Error())

	// Test that List returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	_, err = entry.List(ctx)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that List properly decodes the entries from stdout
	stdout := "[" +
		"{\"name\":\"foo\",\"methods\":[\"list\"],\"type_id\":\"bar\"}" +
		"]"
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		entryBase := plugin.NewEntry("foo")
		expectedEntries := []plugin.Entry{
			&pluginEntry{
				EntryBase: entryBase,
				methods: map[string]methodInfo{
					"list": methodInfo{
						signature: plugin.DefaultSignature,
					},
				},
				script:       entry.script,
				schemaGraphs: entry.schemaGraphs,
				rawTypeID:    "bar",
			},
		}

		suite.Equal(expectedEntries, entries)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestRead() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "read", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Read returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.Read(ctx)
	suite.EqualError(err, mockErr.Error())

	// Test that Read returns the invocation's stdout
	stdout := "foo"
	mockInvokeAndWait([]byte(stdout), nil)
	content, err := entry.Read(ctx)
	if suite.NoError(err) {
		expectedContent := []byte(stdout)
		suite.Equal(expectedContent, content)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestBlockRead() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "read", entry, "10", "0").Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then BlockRead returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.BlockRead(ctx, 10, 0)
	suite.EqualError(err, mockErr.Error())

	// Test that BlockRead returns the invocation's stdout
	stdout := "foo"
	mockInvokeAndWait([]byte(stdout), nil)
	content, err := entry.BlockRead(ctx, 10, 0)
	if suite.NoError(err) {
		expectedContent := []byte(stdout)
		suite.Equal(expectedContent, content)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestListReadWithMethodResults() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
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
[{"name": "foo", "methods": [
	["list", [
		{"name": "bar", "methods": [["read", "` + content + `"]]}
	]]
]}]`)
	}
	someContent := "some content"
	mockInvokeAndWait([]byte(stdoutFn(someContent)), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries))
		if suite.Equal([]string{"list"}, plugin.SupportedActionsOf(entries[0])) {
			children, err := entries[0].(plugin.Parent).List(ctx)
			if suite.NoError(err) {
				suite.Equal(1, len(children))
				attr := plugin.Attributes(children[0])
				if suite.True(attr.HasSize()) {
					suite.Equal(uint64(len(someContent)), attr.Size())
				}

				if suite.Equal([]string{"read"}, plugin.SupportedActionsOf(children[0])) {
					content, err := children[0].(plugin.Readable).Read(ctx)
					suite.NoError(err)
					suite.Equal(someContent, string(content))
				}
			}
		}
	}
}

type mockedInvocation struct {
	Command
	mock.Mock
}

func (m *mockedInvocation) RunAndWait(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func (m *mockedInvocation) Stdout() *bytes.Buffer {
	return m.Called().Get(0).(*bytes.Buffer)
}

func (m *mockedInvocation) Stderr() *bytes.Buffer {
	return m.Called().Get(0).(*bytes.Buffer)
}

var _ = invocation(&mockedInvocation{})

func (suite *ExternalPluginEntryTestSuite) TestWrite() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")
	data := []byte("something to write")

	ctx := context.Background()
	mockRunAndWait := func(err error) {
		mockInv := &mockedInvocation{Command: NewCommand(ctx, "")}
		mockScript.On("NewInvocation", ctx, "write", entry, []string(nil)).Return(mockInv).Once()
		mockInv.On("RunAndWait", ctx).Return(err).Once()
	}

	// Test that if RunAndWait errors, then Write returns its error
	mockErr := fmt.Errorf("execution error")
	mockRunAndWait(mockErr)
	err := entry.Write(ctx, data)
	suite.EqualError(err, mockErr.Error())

	// Test that invocation succeeds
	mockRunAndWait(nil)
	err = entry.Write(ctx, data)
	suite.NoError(err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeWithErrors() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that List validates prefetched read
	stdout := `[{"name": "foo", "methods": [["read", [1, 2]]]}]`
	mockInvokeAndWait([]byte(stdout), nil)
	_, err := entry.List(ctx)
	suite.EqualError(err, "Read method must provide a string, not [1, 2]")

	// Test that List validates prefetched list
	stdout = `[{"name": "foo", "methods": [["list", {"name": "bar"}]]}]`
	mockInvokeAndWait([]byte(stdout), nil)
	_, err = entry.List(ctx)
	suite.EqualError(err, `implementation of list must conform to `+
		`[{"name":"entry1","methods":["list"]},{"name":"entry2","methods":["list"]}], not {"name": "bar"}`)

	// Test that List validates block readable tag
	stdout = `[{"name": "foo", "methods": [["read", true]]}]`
	mockInvokeAndWait([]byte(stdout), nil)
	_, err = entry.List(ctx)
	suite.NoError(err)
}

func (suite *ExternalPluginEntryTestSuite) TestMetadata_NotImplemented() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		script:    mockScript,
	}
	expectedMeta := plugin.JSONObject{"foo": "bar"}
	entry.SetPartialMetadata(expectedMeta)

	// If metadata is not implemented, then e.Metadata should return
	// EntryBase#Metadata, which returns the partial metadata.
	meta, err := entry.Metadata(context.Background())
	if suite.NoError(err) {
		suite.Equal(expectedMeta, meta)
	}
	// Make sure that Wash did not shell out to the plugin script
	mockScript.AssertNotCalled(suite.T(), "InvokeAndWait")
}

func (suite *ExternalPluginEntryTestSuite) TestMetadata_Implemented() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		methods:   map[string]methodInfo{"metadata": methodInfo{}},
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
	suite.EqualError(err, mockErr.Error())

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
		expectedMetadata := plugin.JSONObject{"key": "value"}
		suite.Equal(expectedMetadata, metadata)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSignal() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		methods:   map[string]methodInfo{"signal": methodInfo{}},
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(signal string, stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "signal", entry, signal).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Signal returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait("start", []byte{}, mockErr)
	err := entry.Signal(ctx, "start")
	suite.EqualError(err, mockErr.Error())

	// Test that Signal properly signals the entry
	mockInvokeAndWait("start", []byte{}, nil)
	err = entry.Signal(ctx, "start")
	if suite.NoError(err) {
		mockScript.AssertExpectations(suite.T())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDelete() {
	mockScript := &mockPluginScript{path: "plugin_script"}
	entry := &pluginEntry{
		EntryBase: plugin.NewEntry("foo"),
		methods:   map[string]methodInfo{"delete": methodInfo{}},
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "delete", entry).Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then Delete returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.Delete(ctx)
	suite.EqualError(err, mockErr.Error())

	// Test that Delete returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	_, err = entry.Delete(ctx)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Delete properly deletes the entry and decodes the result
	// from stdout
	mockInvokeAndWait([]byte("true"), nil)
	deleted, err := entry.Delete(ctx)
	if suite.NoError(err) {
		suite.True(deleted)
	}
}

// TODO: Add tests for stdoutStreamer, Stream and Exec
// once the API for Stream and Exec's at a more stable
// state.

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotAJSONObject() {
	entry := &pluginEntry{}
	entry.SetTestID("/fooPlugin")
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), []byte("[]"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfAnEmptyJSONObject() {
	entry := &pluginEntry{}
	entry.SetTestID("/fooPlugin")
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), []byte("{}"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfTypeIDNotPresent() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")
	stdout := []byte(`
{
	"bar": "baz"
}`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("foo.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnMalformedSchema() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")

	// Error should indicate that foo's schema is not a JSON object.
	stdout := []byte(`
{
	"foo": "fooSchema",
	"bar": {}
}`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("object.*foo.*fooSchema", err)

	// Error should indicate that "foo"'s label is malformed.
	stdout = []byte(`
{
	"foo": {
		"label": 5
	},
	"bar": {}
}`)
	_, err = unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
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
	_, err = unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp(`\[\]string`, err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfLabelNotProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")

	stdout := []byte(`
{
	"foo":{}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("label.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMethodsNotProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel"
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("methods.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotParentAndChildrenProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"],
		"children": ["barTypeID"]
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("entry.*children.*not.*parent", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfParentAndChildrenNotProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["list"],
		"children": []
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("parent.*entries.*children", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotSignalableAndSignalsProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": [],
		"signals": [
		  {"name": "foo", "description": "bar"}
		]
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("entry.*signals.*not.*signalable", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfSignalableAndSignalsNotProvided() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["signal"],
		"signals": []
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("signalable.*entries.*signal", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMissingChildSchema() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["list"],
		"children": ["bar"]
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("bar.*schema.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfSchemaIncludesDanglingTypeIDs() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

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
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	// Need to do several regexp checks here b/c this error
	// message is generated by iterating over a map's keys.
	// Map keys are iterated in random order.
	suite.Regexp("bar", err)
	suite.Regexp("baz", err)
	suite.Regexp("associated", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidPartialMetadataSchema() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

	stdout := []byte(`
{
	"foo":{
		"label": "fooLabel",
		"methods": ["read"],
		"partial_metadata_schema": {
			"type": "array"
		}
	}
}
`)
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("invalid.*partial.*metadata.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidMetadataSchema() {
	entry := &pluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

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
	_, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	suite.Regexp("invalid.*metadata.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ValidInput() {
	// This test tests unmarshalSchemaGraph by ensuring that the returned graph
	// can marshal back into its original form (with the "methods" key replaced
	// by the "actions" key)
	stdout := readSchemaFixture(suite.Suite, "externalPluginSchema")

	entry := &pluginEntry{
		rawTypeID: "aws.Root",
	}
	entry.SetTestID("fooPlugin")
	actualGraph, err := unmarshalSchemaGraph(pluginName(entry), rawTypeID(entry), stdout)
	if suite.NoError(err) {
		// Check that the first key is aws.Root
		it := actualGraph.Iterator()
		it.First()
		suite.Equal(it.Key(), plugin.TypeID(entry))

		// Now check that all of stdout was successfully unmarshalled.
		stdout = bytes.ReplaceAll(stdout, []byte("methods"), []byte("actions"))
		stdout = bytes.ReplaceAll(stdout, []byte("aws."), []byte("fooPlugin::aws."))
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
