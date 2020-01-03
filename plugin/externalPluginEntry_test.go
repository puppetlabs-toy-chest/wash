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
	retValues := m.Called(ctx, method, entry, args)
	return retValues.Get(0).(invocation)
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

	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("name", err)
	decodedEntry.Name = "decodedEntry"

	_, err = decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("methods", err)
	decodedEntry.Methods = []interface{}{"list"}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Equal(1, len(entry.methods))
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"].result)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryExtraFields() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "stream"},
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.Equal(defaultSignature, entry.methods["list"].signature)
		suite.Nil(entry.methods["list"].result)
		suite.Contains(entry.methods, "stream")
		suite.Equal(defaultSignature, entry.methods["stream"].signature)
		suite.Nil(entry.methods["stream"].result)
		suite.False(entry.isPrefetched())

		methods := entry.supportedMethods()
		suite.Equal(2, len(methods))
		suite.Contains(methods, "list")
		suite.Contains(methods, "stream")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntry_SupportsEmptyMethodsArray() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{},
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithMethodResults() {
	childEntry := map[string]interface{}{"name": "foo", "methods": []string{"read"}}
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{[]interface{}{"list", []interface{}{childEntry}}, "read"},
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.Equal(defaultSignature, entry.methods["list"].signature)
		suite.NotNil(entry.methods["list"].result)
		suite.Contains(entry.methods, "read")
		suite.Nil(entry.methods["read"].result)
		suite.True(entry.isPrefetched())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryMethodTuple_Read() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{[]interface{}{"read", true}},
	}

	type testCase struct {
		data              interface{}
		expectedSignature methodSignature
		expectedResult    interface{}
	}
	testCases := []testCase{
		testCase{true, blockReadableSignature, nil},
		testCase{false, defaultSignature, nil},
		testCase{"foo", defaultSignature, "foo"},
	}
	for _, testCase := range testCases {
		decodedEntry.Methods = []interface{}{[]interface{}{"read", testCase.data}}
		entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
		if suite.NoError(err) {
			suite.NotNil(entry.methods["read"])
			suite.Equal(testCase.expectedSignature, entry.methods["read"].signature)
			suite.Equal(testCase.expectedResult, entry.methods["read"].result)
		}
	}
}

func newMockDecodedEntry(name string) decodedExternalPluginEntry {
	return decodedExternalPluginEntry{
		Name:    name,
		Methods: []interface{}{"list"},
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
		expectedTTLs := NewEntry("foo").ttl
		expectedTTLs[ListOp] = decodedEntry.CacheTTLs.List * time.Second
		suite.Equal(expectedTTLs, entry.EntryBase.ttl)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSlashReplacer() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.SlashReplacer = "a string"
	suite.Panics(
		func() { _, _ = decodedEntry.toExternalPluginEntry(context.Background(), false, false) },
		"e.SlashReplacer: received string a string instead of a character",
	)
	decodedEntry.SlashReplacer = ":"
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.Equal(':', entry.slashReplacer())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithAttributes() {
	decodedEntry := newMockDecodedEntry("name")
	t := time.Now()
	decodedEntry.Attributes.SetCtime(t)
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
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
	}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.False(entry.schemaKnown)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_ImplementsSchema_TypeIDNotIncluded() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("decodedEntry.*implements.*schema.*no.*type.*ID", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaUnknown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	suite.Regexp("decodedEntry.*foo.*implements.*schema.*root", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_DoesNotImplementSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list"},
		TypeID:  "foo",
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*foo.*must.*implement.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_ImplementsSchema_TypeIDNotIncluded() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
	}
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*implements.*schema.*no.*type.*ID", err)
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
	_, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	suite.Regexp("decodedEntry.*foo.*plugin.*roots.*support.*prefetching", err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSchema_SchemaKnown_ImplementsSchema() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "schema"},
		TypeID:  "foo",
	}
	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), true, false)
	if suite.NoError(err) {
		suite.True(entry.schemaKnown)
		suite.Equal("foo", rawTypeID(entry))
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithInaccessibleReason() {
	decodedEntry := decodedExternalPluginEntry{
		Name:               "decodedEntry",
		Methods:            []interface{}{"list", "stream"},
		InaccessibleReason: "permission denied",
	}

	entry, err := decodedEntry.toExternalPluginEntry(context.Background(), false, false)
	if suite.NoError(err) {
		suite.True(entry.isInaccessible())
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
	suite.Equal(decodedTTLs.Read*time.Second, entry.getTTLOf(ReadOp))
	suite.Equal(decodedTTLs.Metadata*time.Second, entry.getTTLOf(MetadataOp))
}

func mockInvocation(stdout []byte) invocation {
	return &invocationImpl{Command: internal.NewCommand(context.Background(), ""), stdout: *bytes.NewBuffer(stdout)}
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
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
	}
	entry.SetTestID("/fooPlugin")

	suite.Panics(
		func() { _, _ = entry.schema() },
		"e.Schema(): entry schemas were prefetched, but no schema graph was provided for /foo (type ID fooTypeID)",
	)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsTheSchema() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/fooPlugin")
	graph := linkedhashmap.New()
	graph.Put(
		TypeID(entry),
		entrySchema{
			Actions: []string{"schema"},
		},
	)
	entry.schemaGraphs[TypeID(entry)] = graph

	s, err := entry.schema()
	if suite.NoError(err) {
		suite.Equal(entry.schemaGraphs[TypeID(entry)], s.graph)
		suite.Equal(s.entrySchema.Actions, []string{"schema"})
		// Make sure that Wash did not shell out to the plugin script
		mockScript.AssertNotCalled(suite.T(), "InvokeAndWait")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_Prefetched_ReturnsErrorIfSchemaAndInstanceMethodsDontMatch() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
			"read":   methodInfo{},
		},
		schemaGraphs: make(map[string]*linkedhashmap.Map),
		script:       mockScript,
	}
	entry.SetTestID("/fooPlugin")
	graph := linkedhashmap.New()
	graph.Put(
		TypeID(entry),
		entrySchema{
			Actions: []string{"list", "exec"},
		},
	)
	entry.schemaGraphs[TypeID(entry)] = graph

	_, err := entry.schema()
	suite.Regexp("schema.*methods.*exec.*list.*instance.*methods.*read.*schema", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfInvocationFails() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		script: mockScript,
	}
	entry.SetTestID("/fooPlugin")

	invokeErr := fmt.Errorf("invocation failed")
	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte{}), invokeErr).Once()
	_, err := entry.schema()
	suite.Regexp("foo.*fooTypeID.*invocation.*failed", err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_ReturnsErrorIfUnmarshallingFails() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		rawTypeID: "fooTypeID",
		methods: map[string]methodInfo{
			"schema": methodInfo{},
		},
		script: mockScript,
	}
	entry.SetTestID("/fooPlugin")

	mockScript.OnInvokeAndWait(mock.Anything, "schema", entry).Return(mockInvocation([]byte("\"foo\"")), nil).Once()
	_, err := entry.schema()
	suite.Regexp("/foo.*fooTypeID", err)
	suite.Regexp(regexp.QuoteMeta(schemaFormat), err)
}

func (suite *ExternalPluginEntryTestSuite) TestSchema_NotPrefetched_SuccessfulInvocation_ReturnsSchema() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
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
		"meta_attribute_schema": {
			"type": "object"
		},
		"metadata_schema": null
	},
	"baz.barTypeID": {
		"label": "barEntry",
		"methods": ["list"],
		"children": ["baz.barTypeID"],
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
			stdout = strings.ReplaceAll(stdout, "baz.", "fooPlugin::baz.")
			suite.JSONEq(stdout, string(schemaJSON))
			suite.Equal(schema.Actions, []string{"list"})
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
	suite.EqualError(mockErr, err.Error())

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
		entryBase := NewEntry("foo")
		expectedEntries := []Entry{
			&externalPluginEntry{
				EntryBase: entryBase,
				methods: map[string]methodInfo{
					"list": methodInfo{
						signature: defaultSignature,
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

	// Test that if InvokeAndWait errors, then Read returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.Read(ctx)
	suite.EqualError(mockErr, err.Error())

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
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "read", entry, "10", "0").Return(mockInvocation(stdout), err).Once()
	}

	// Test that if InvokeAndWait errors, then blockRead returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	_, err := entry.blockRead(ctx, 10, 0)
	suite.EqualError(mockErr, err.Error())

	// Test that blockRead returns the invocation's stdout
	stdout := "foo"
	mockInvokeAndWait([]byte(stdout), nil)
	content, err := entry.blockRead(ctx, 10, 0)
	if suite.NoError(err) {
		expectedContent := []byte(stdout)
		suite.Equal(expectedContent, content)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestListReadWithMethodResults() {
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
		if suite.Equal([]string{"list"}, SupportedActionsOf(entries[0])) {
			children, err := entries[0].(Parent).List(ctx)
			if suite.NoError(err) {
				suite.Equal(1, len(children))
				attr := Attributes(children[0])
				if suite.True(attr.HasSize()) {
					suite.Equal(uint64(len(someContent)), attr.Size())
				}

				if suite.Equal([]string{"read"}, SupportedActionsOf(children[0])) {
					content, err := children[0].(Readable).Read(ctx)
					suite.NoError(err)
					suite.Equal(someContent, string(content))
				}
			}
		}
	}
}

type mockedInvocation struct {
	internal.Command
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
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")
	data := []byte("something to write")

	ctx := context.Background()
	mockRunAndWait := func(err error) {
		mockInv := &mockedInvocation{Command: internal.NewCommand(ctx, "")}
		mockScript.On("NewInvocation", ctx, "write", entry, []string(nil)).Return(mockInv).Once()
		mockInv.On("RunAndWait", ctx).Return(err).Once()
	}

	// Test that if RunAndWait errors, then Write returns its error
	mockErr := fmt.Errorf("execution error")
	mockRunAndWait(mockErr)
	err := entry.Write(ctx, data)
	suite.EqualError(mockErr, err.Error())

	// Test that invocation succeeds
	mockRunAndWait(nil)
	err = entry.Write(ctx, data)
	suite.NoError(err)
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
	stdout := `[{"name": "foo", "methods": [
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
			`[{"name":"entry1","methods":["list"]},{"name":"entry2","methods":["list"]}], not map[name:bar]`)

		_, err = entries[0].(Readable).Read(ctx)
		suite.EqualError(err, "Read method must provide a string, not [1 2]")
		// TODO: Add a test for block readable here
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

func (suite *ExternalPluginEntryTestSuite) TestSignal() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
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
	suite.EqualError(mockErr, err.Error())

	// Test that Signal properly signals the entry
	mockInvokeAndWait("start", []byte{}, nil)
	err = entry.Signal(ctx, "start")
	if suite.NoError(err) {
		mockScript.AssertExpectations(suite.T())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDelete() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
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
	suite.EqualError(mockErr, err.Error())

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
	entry := &externalPluginEntry{}
	entry.SetTestID("/fooPlugin")
	_, err := unmarshalSchemaGraph(entry, []byte("[]"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfAnEmptyJSONObject() {
	entry := &externalPluginEntry{}
	entry.SetTestID("/fooPlugin")
	_, err := unmarshalSchemaGraph(entry, []byte("{}"))
	suite.Regexp("non-empty.*object", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfTypeIDNotPresent() {
	entry := &externalPluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")
	stdout := []byte(`
{
	"bar": "baz"
}`)
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("foo.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnMalformedSchema() {
	entry := &externalPluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")

	// Error should indicate that foo's schema is not a JSON object.
	stdout := []byte(`
{
	"foo": "fooSchema",
	"bar": {}
}`)
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("object.*foo.*fooSchema", err)

	// Error should indicate that "foo"'s label is malformed.
	stdout = []byte(`
{
	"foo": {
		"label": 5
	},
	"bar": {}
}`)
	_, err = unmarshalSchemaGraph(entry, stdout)
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
	_, err = unmarshalSchemaGraph(entry, stdout)
	suite.Regexp(`\[\]string`, err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfLabelNotProvided() {
	entry := &externalPluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("/fooPlugin")

	stdout := []byte(`
{
	"foo":{}
}
`)
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("label.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMethodsNotProvided() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("methods.*provided", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotParentAndChildrenProvided() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("entry.*children.*not.*parent", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfParentAndChildrenNotProvided() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("parent.*entries.*children", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfNotSignalableAndSignalsProvided() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("entry.*signals.*not.*signalable", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfSignalableAndSignalsNotProvided() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("signalable.*entries.*signal", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfMissingChildSchema() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("bar.*schema.*missing", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsIfSchemaIncludesDanglingTypeIDs() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	// Need to do several regexp checks here b/c this error
	// message is generated by iterating over a map's keys.
	// Map keys are iterated in random order.
	suite.Regexp("bar", err)
	suite.Regexp("baz", err)
	suite.Regexp("associated", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidMetaAttributeSchema() {
	entry := &externalPluginEntry{
		rawTypeID: "foo",
	}
	entry.SetTestID("fooPlugin")

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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("invalid.*meta.*attribute.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ErrorsOnInvalidMetadataSchema() {
	entry := &externalPluginEntry{
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
	_, err := unmarshalSchemaGraph(entry, stdout)
	suite.Regexp("invalid.*metadata.*object.*schema.*array", err)
}

func (suite *ExternalPluginEntryTestSuite) TestUnmarshalSchemaGraph_ValidInput() {
	// This test tests unmarshalSchemaGraph by ensuring that the returned graph
	// can marshal back into its original form (with the "methods" key replaced
	// by the "actions" key)
	stdout := readSchemaFixture(suite.Suite, "externalPluginSchema")

	entry := &externalPluginEntry{
		rawTypeID: "aws.Root",
	}
	entry.SetTestID("fooPlugin")
	actualGraph, err := unmarshalSchemaGraph(entry, stdout)
	if suite.NoError(err) {
		// Check that the first key is aws.Root
		it := actualGraph.Iterator()
		it.First()
		suite.Equal(it.Key(), TypeID(entry))

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
