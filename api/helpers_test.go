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

type mockParent struct {
	plugin.EntryBase
	entries []plugin.Entry
}

func (g *mockParent) List(context.Context) ([]plugin.Entry, error) {
	return g.entries, nil
}

func (g *mockParent) ChildSchemas() []plugin.EntrySchema {
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

func (suite *HelpersTestSuite) TestFindEntry() {
	type testcase struct {
		segments      []string
		expectedEntry string
		expectedErr   error
	}
	runTestCase := func(parent plugin.Parent, c testcase) {
		got, err := findEntry(context.Background(), parent, c.segments)
		if c.expectedEntry != "" && suite.NotNil(got) {
			suite.Equal(c.expectedEntry, plugin.CName(got))
		} else {
			suite.Nil(got)
		}
		if c.expectedErr == nil {
			suite.Nil(err)
		} else {
			suite.Equal(c.expectedErr, err)
		}
	}

	foo := plugin.NewEntryBase()
	foo.SetName("foo/bar")
	parent := &mockParent{plugin.NewEntryBase(), []plugin.Entry{&foo}}
	parent.SetName("root")
	parent.SetTestID("/root")
	parent.DisableDefaultCaching()
	for _, c := range []testcase{
		{[]string{"not found"}, "", entryNotFoundResponse("not found", "The not found entry does not exist")},
		{[]string{"foo#bar"}, "foo#bar", nil},
		{[]string{"foo#bar", "bar"}, "", entryNotFoundResponse("foo#bar/bar", "The entry foo#bar is not a parent")},
	} {
		runTestCase(parent, c)
	}

	baz := plugin.NewEntryBase()
	baz.SetName("baz")
	nestedParent := &mockParent{plugin.NewEntryBase(), []plugin.Entry{&baz}}
	nestedParent.SetName("bar")
	nestedParent.DisableDefaultCaching()
	parent.entries = append(parent.entries, nestedParent)
	for _, c := range []testcase{
		{[]string{"bar"}, "bar", nil},
		{[]string{"bar", "foo"}, "", entryNotFoundResponse("bar/foo", "The foo entry does not exist in the bar parent")},
		{[]string{"bar", "baz"}, "baz", nil},
	} {
		runTestCase(parent, c)
	}

	// Finally, test the duplicate cname error response
	duplicateFoo := plugin.NewEntryBase()
	duplicateFoo.SetName("foo#bar")
	parent.entries = append(parent.entries, &duplicateFoo)
	expectedErr := plugin.DuplicateCNameErr{
		ParentID:                 plugin.ID(parent),
		FirstChildName:           foo.Name(),
		FirstChildSlashReplacer:  '#',
		SecondChildName:          duplicateFoo.Name(),
		SecondChildSlashReplacer: '#',
		CName:                    "foo#bar",
	}
	runTestCase(
		parent,
		testcase{[]string{"foo#bar"}, "", duplicateCNameResponse(expectedErr)},
	)
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

func (m *mockRoot) ChildSchemas() []plugin.EntrySchema {
	return []plugin.EntrySchema{
		plugin.EntrySchema{
			TypeID: "mockEntry",
		},
	}
}

func getRequest(ctx context.Context, path string) *http.Request {
	return (&http.Request{URL: &url.URL{RawQuery: url.Values{"path": []string{path}}.Encode()}}).WithContext(ctx)
}

func (suite *HelpersTestSuite) TestGetEntryFromPath() {
	reg := plugin.NewRegistry()
	plug := &mockRoot{EntryBase: plugin.NewEntryBase()}
	plug.SetName("mine")
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

	file := plugin.NewEntryBase()
	file.SetName("a file")
	file.SetTestID("/mine/a file")
	plug.On("List", mock.Anything).Return([]plugin.Entry{&file}, nil)

	entry, path, err = getEntryFromRequest(getRequest(ctx, mountpoint+"/mine/a file"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/mine/a file", path)
		suite.Equal(file.Name(), plugin.Name(entry))
	}
	plug.AssertExpectations(suite.T())

	plug.On("List", mock.Anything).Return([]plugin.Entry{&file}, nil)
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
