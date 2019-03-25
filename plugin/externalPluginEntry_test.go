package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

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

func (m *mockExternalPluginScript) InvokeAndWait(ctx context.Context, args ...string) ([]byte, error) {
	retValues := m.Called(ctx, args)
	return retValues.Get(0).([]byte), retValues.Error(1)
}

// We make ctx an interface{} so that this method could
// be used when the caller generates a context using e.g.
// context.Background()
func (m *mockExternalPluginScript) OnInvokeAndWait(ctx interface{}, args ...string) *mock.Call {
	return m.On("InvokeAndWait", ctx, args)
}

type ExternalPluginEntryTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginEntryTestSuite) EqualTimeAttr(expected time.Time, actual time.Time, msgAndArgs ...interface{}) {
	suite.WithinDuration(expected, actual, 1*time.Second, msgAndArgs...)
}

func (suite *ExternalPluginEntryTestSuite) TestUnixSecondsToTimeAttr() {
	suite.EqualTimeAttr(time.Time{}, unixSecondsToTimeAttr(0))

	t := time.Now()
	suite.EqualTimeAttr(t, unixSecondsToTimeAttr(t.Unix()))
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeAttributes() {
	atime := time.Now()
	mtime := time.Now()
	ctime := time.Now()

	decodedAttributes := decodedAttributes{
		Atime: atime.Unix(),
		Mtime: mtime.Unix(),
		Ctime: ctime.Unix(),
		Size:  10,
		Valid: 1 * time.Second,
	}

	// Test that the attributes are correctly decoded
	attributes, err := decodedAttributes.toAttributes()
	if suite.NoError(err) {
		suite.EqualTimeAttr(atime, attributes.Atime)
		suite.EqualTimeAttr(mtime, attributes.Mtime)
		suite.EqualTimeAttr(ctime, attributes.Ctime)
		suite.Equal(uint64(10), attributes.Size)
		suite.Equal(
			os.FileMode(0),
			attributes.Mode,
			"Expected the decoded attributes to have no mode field",
		)
	}

	// Test that the mode is correctly decoded
	decodedAttributes.Mode = "0xff"
	attributes, err = decodedAttributes.toAttributes()
	if suite.NoError(err) {
		suite.Equal(os.FileMode(255), attributes.Mode)
	}

	// Test that toAttributes() returns an error when the mode
	// cannot be decoded
	decodedAttributes.Mode = "not a number"
	attributes, err = decodedAttributes.toAttributes()
	suite.Error(err)
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryRequiredFields() {
	decodedEntry := decodedExternalPluginEntry{}

	_, err := decodedEntry.toExternalPluginEntry()
	suite.Regexp(regexp.MustCompile("name"), err)
	decodedEntry.Name = "decodedEntry"

	_, err = decodedEntry.toExternalPluginEntry()
	suite.Regexp(regexp.MustCompile("action"), err)
	decodedEntry.SupportedActions = []string{"list"}

	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(decodedEntry.Name, entry.name)
		suite.Equal(decodedEntry.SupportedActions, entry.supportedActions)
	}
}

func newMockDecodedEntry(name string) decodedExternalPluginEntry {
	return decodedExternalPluginEntry{
		Name:             name,
		SupportedActions: []string{"list"},
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
		expectedTTLs := NewEntry("mock").ttl
		expectedTTLs[List] = decodedEntry.CacheTTLs.List * time.Second
		suite.Equal(expectedTTLs, entry.EntryBase.ttl)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithSlashReplacementChar() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.SlashReplacementChar = "a string"
	suite.Panics(
		func() { _, _ = decodedEntry.toExternalPluginEntry() },
		"e.SlashReplacementChar: received string a string instead of a character",
	)
	decodedEntry.SlashReplacementChar = ":"
	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(':', entry.slashReplacementChar())
	}
}

func (suite *ExternalPluginEntryTestSuite) TestDecodeExternalPluginEntryWithAttributes() {
	decodedEntry := newMockDecodedEntry("name")
	decodedEntry.Attributes = decodedAttributes{Size: 10}
	entry, err := decodedEntry.toExternalPluginEntry()
	if suite.NoError(err) {
		suite.Equal(Attributes{Size: 10}, entry.attr)
	}

	decodedEntry.Attributes = decodedAttributes{Mode: "invalid mode"}
	_, err = decodedEntry.toExternalPluginEntry()
	suite.Error(err)
}

func (suite *ExternalPluginEntryTestSuite) TestSetCacheTTLs() {
	decodedTTLs := decodedCacheTTLs{
		List:     10,
		Open:     15,
		Metadata: 20,
	}

	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
	}
	entry.setCacheTTLs(decodedTTLs)

	suite.Equal(decodedTTLs.List*time.Second, entry.getTTLOf(List))
	suite.Equal(decodedTTLs.Open*time.Second, entry.getTTLOf(Open))
	suite.Equal(decodedTTLs.Metadata*time.Second, entry.getTTLOf(Metadata))
}

// TODO: There's a bit of duplication between TestList, TestOpen,
// and TestMetadata that could be refactored. Not worth doing right
// now since the refactor may make the tests harder to understand,
// but could be worth considering later if we add similarly structured
// actions.

func (suite *ExternalPluginEntryTestSuite) TestList() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "list", entry.id(), entry.state).Return(stdout, err).Once()
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
		"{\"name\":\"foo\",\"supported_actions\":[\"list\"]}" +
		"]"
	mockInvokeAndWait([]byte(stdout), nil)
	entries, err := entry.List(ctx)
	if suite.NoError(err) {
		expectedEntries := []Entry{
			&externalPluginEntry{
				EntryBase:        NewEntry("foo"),
				supportedActions: []string{"list"},
				script:           entry.script,
			},
		}

		suite.Equal(expectedEntries, entries)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestOpen() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "read", entry.id(), entry.state).Return(stdout, err).Once()
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

func (suite *ExternalPluginEntryTestSuite) TestMetadata() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	entry := externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}
	entry.SetTestID("/foo")

	ctx := context.Background()
	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(ctx, "metadata", entry.id(), entry.state).Return(stdout, err).Once()
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
		expectedMetadata := MetadataMap{"key": "value"}
		suite.Equal(expectedMetadata, metadata)
	}
}

func (suite *ExternalPluginEntryTestSuite) TestAttr() {
	entry := externalPluginEntry{attr: Attributes{Size: 10}}
	attr, _ := entry.Attr(context.Background())
	suite.Equal(entry.attr, attr)
}

// TODO: Add tests for stdoutStreamer, Stream and Exec
// once the API for Stream and Exec's at a more stable
// state.

func TestExternalPluginEntry(t *testing.T) {
	suite.Run(t, new(ExternalPluginEntryTestSuite))
}
