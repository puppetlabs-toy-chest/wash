package api

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockEntry struct {
	plugin.EntryBase
}

func newMockEntry(name string) *mockEntry {
	return &mockEntry{
		EntryBase: plugin.NewEntry(name),
	}
}

func (e *mockEntry) Schema() *plugin.EntrySchema {
	return nil
}

type HelpersTestSuite struct {
	suite.Suite
}

func (suite *HelpersTestSuite) SetupSuite() {
	plugin.SetTestCache(newMockCache())
}

func (suite *HelpersTestSuite) TearDownSuite() {
	plugin.UnsetTestCache()
}

type mockRoot struct {
	plugin.EntryBase
	mock.Mock
}

func (m *mockRoot) Init(map[string]interface{}) error {
	return nil
}

func (m *mockRoot) List(ctx context.Context) ([]plugin.Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}

func (m *mockRoot) Schema() *plugin.EntrySchema {
	return nil
}

func (m *mockRoot) ChildSchemas() []*plugin.EntrySchema {
	return nil
}

func (m *mockRoot) WrappedTypes() plugin.SchemaMap {
	return nil
}

func getRequest(ctx context.Context, path string) *http.Request {
	return (&http.Request{URL: &url.URL{RawQuery: url.Values{"path": []string{path}}.Encode()}}).WithContext(ctx)
}

func (suite *HelpersTestSuite) TestGetEntryFromPath() {
	reg := plugin.NewRegistry()
	plug := &mockRoot{EntryBase: plugin.NewEntry("mine")}
	plug.SetTestID("/mine")
	suite.NoError(reg.RegisterPlugin(plug, map[string]interface{}{}))
	ctx := context.WithValue(context.Background(), pluginRegistryKey, reg)

	mountpoint := "/mountpoint"
	ctx = context.WithValue(ctx, mountpointKey, mountpoint)

	_, _, err := getEntryFromRequest(getRequest(ctx, "relative"))
	suite.Error(relativePathResponse("relative"), err)

	// TODO: Add tests for non-Wash entries (i.e. for apifs)

	entry, path, err := getEntryFromRequest(getRequest(ctx, mountpoint))
	if suite.Nil(err) {
		suite.Equal(mountpoint, path)
		suite.Equal(reg.Name(), plugin.Name(entry))
	}

	entry, path, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/", path)
		suite.Equal(reg.Name(), plugin.Name(entry))
	}

	entry, path, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/mine"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/mine", path)
		suite.Equal(plug.Name(), plugin.Name(entry))
	}

	_, _, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/yours"))
	suite.Error(err)

	file := newMockEntry("a file")
	file.SetTestID("/mine/a file")
	plug.On("List", mock.Anything).Return([]plugin.Entry{file}, nil)

	entry, path, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/mine/a file"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/mine/a file", path)
		suite.Equal(file.Name(), plugin.Name(entry))
	}
	plug.AssertExpectations(suite.T())

	plug.On("List", mock.Anything).Return([]plugin.Entry{file}, nil)
	_, _, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/mine/a dir"))
	suite.Error(err)
	plug.AssertExpectations(suite.T())
}

func (suite *HelpersTestSuite) TestGetScalarParam() {
	var u url.URL
	suite.Empty(getScalarParam(&u, "param"))

	u.RawQuery = "param=hello"
	suite.Equal("hello", getScalarParam(&u, "param"))

	u.RawQuery = "param=hello&param=goodbye"
	suite.Equal("goodbye", getScalarParam(&u, "param"))
}

func (suite *HelpersTestSuite) TestGetBoolParam() {
	var u url.URL
	for query, expect := range map[string]bool{"": false, "param=true": true, "param=false": false} {
		u.RawQuery = query
		val, err := getBoolParam(&u, "param")
		suite.Nil(err)
		suite.Equal(expect, val)
	}

	u.RawQuery = "param=other"
	_, err := getBoolParam(&u, "param")
	suite.Error(err)
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
