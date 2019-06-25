package plugin

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

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
) ([]byte, error) {
	retValues := m.Called(ctx, method, entry, args)
	return retValues.Get(0).([]byte), retValues.Error(1)
}

func (m *mockExternalPluginScript) NewInvocation(
	ctx context.Context,
	method string,
	entry *externalPluginEntry,
	args ...string,
) *internal.Command {
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

	_, err := decodedEntry.toExternalPluginEntry()
	suite.Regexp(regexp.MustCompile("name"), err)
	decodedEntry.Name = "decodedEntry"

	_, err = decodedEntry.toExternalPluginEntry()
	suite.Regexp(regexp.MustCompile("methods"), err)
	decodedEntry.Methods = []interface{}{"list"}

	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Equal(1, len(entry.methods))
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"])
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryExtraFields() {
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{"list", "stream"},
	}

	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.Nil(entry.methods["list"])
		suite.Contains(entry.methods, "stream")
		suite.Nil(entry.methods["stream"])

		methods := entry.supportedMethods()
		suite.Equal(2, len(methods))
		suite.Contains(methods, "list")
		suite.Contains(methods, "stream")
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithMethodResults() {
	childEntry := map[string]interface{}{"name": "foo", "methods": []string{"read"}}
	decodedEntry := decodedExternalPluginEntry{
		Name:    "decodedEntry",
		Methods: []interface{}{[]interface{}{"list", []interface{}{childEntry}}, "read"},
	}

	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name())
		suite.Contains(entry.methods, "list")
		suite.NotNil(entry.methods["list"])
		suite.Contains(entry.methods, "read")
		suite.Nil(entry.methods["read"])
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
	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(decodedEntry.State, entry.state)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithCacheTTLs() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.CacheTTLs = decodedCacheTTLs{List: 1}
	entry, err := decodedEntry.toExternalPluginEntry()
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
		func() { _, _ = decodedEntry.toExternalPluginEntry() },
		"e.SlashReplacer: received string a string instead of a character",
	)
	decodedEntry.SlashReplacer = ":"
	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(':', entry.slashReplacer())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithAttributes() {
	decodedEntry := newMockDecodedEntry("name")
	t := time.Now()
	decodedEntry.Attributes.SetCtime(t)
	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		expectedAttr := EntryAttributes{}
		expectedAttr.SetCtime(t)
		suite.Equal(expectedAttr, entry.attr)
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

// TODO: There's a bit of duplication between TestList, TestOpen,
// and TestMetadata that could be refactored. Not worth doing right
// now since the refactor may make the tests harder to understand,
// but could be worth considering later if we add similarly structured
// methods.

func (suite *ExternalPluginEntryTestSuite) TestList() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := &externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(stdout, err).Once()
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
		"{\"name\":\"foo\",\"methods\":[\"list\"]}" +
		"]"
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		entryBase := NewEntry("foo")
		expectedEntries := []Entry{
			&externalPluginEntry{
				EntryBase: entryBase,
				methods:   map[string]interface{}{"list": nil},
				script:    entry.script,
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
		mockScript.OnInvokeAndWait(ctx, "read", entry).Return(stdout, err).Once()
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
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(stdout, err).Once()
	}

	// Test that List is invoked when
	stdout := `[{"name": "foo", "methods": [
								["list", [
									{"name": "bar", "methods": [["read", "some content"]]}
									]]
							]}]`
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries))
		if suite.Equal([]string{"list"}, SupportedActionsOf(entries[0])) {
			children, err := entries[0].(Parent).List(ctx)
			if suite.NoError(err) {
				suite.Equal(1, len(children))
				if suite.Equal([]string{"read"}, SupportedActionsOf(children[0])) {
					rdr, err := children[0].(Readable).Open(ctx)
					suite.NoError(err)
					buf := make([]byte, rdr.Size())
					_, err = rdr.ReadAt(buf, 0)
					suite.NoError(err)
					suite.Equal("some content", string(buf))
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
		mockScript.OnInvokeAndWait(ctx, "list", entry).Return(stdout, err).Once()
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
		mockScript.OnInvokeAndWait(ctx, "metadata", entry).Return(stdout, err).Once()
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

func TestExternalPluginEntry(t *testing.T) {
	suite.Run(t, new(ExternalPluginEntryTestSuite))
}
