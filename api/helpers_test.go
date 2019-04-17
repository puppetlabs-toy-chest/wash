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

type mockGroup struct {
	plugin.EntryBase
	entries []plugin.Entry
}

func (g *mockGroup) List(context.Context) ([]plugin.Entry, error) {
	return g.entries, nil
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
	runTestCase := func(grp plugin.Group, c testcase) {
		got, err := findEntry(context.Background(), grp, c.segments)
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

	foo := plugin.NewEntry("foo/bar")
	group := &mockGroup{plugin.NewEntry("root"), []plugin.Entry{&foo}}
	group.SetTestID("/root")
	group.DisableDefaultCaching()
	for _, c := range []testcase{
		{[]string{"not found"}, "", entryNotFoundResponse("not found", "The not found entry does not exist")},
		{[]string{"foo#bar"}, "foo#bar", nil},
		{[]string{"foo#bar", "bar"}, "", entryNotFoundResponse("foo#bar/bar", "The entry foo#bar is not a group")},
	} {
		runTestCase(group, c)
	}

	baz := plugin.NewEntry("baz")
	nestedGroup := &mockGroup{plugin.NewEntry("bar"), []plugin.Entry{&baz}}
	nestedGroup.DisableDefaultCaching()
	group.entries = append(group.entries, nestedGroup)
	for _, c := range []testcase{
		{[]string{"bar"}, "bar", nil},
		{[]string{"bar", "foo"}, "", entryNotFoundResponse("bar/foo", "The foo entry does not exist in the bar group")},
		{[]string{"bar", "baz"}, "baz", nil},
	} {
		runTestCase(group, c)
	}

	// Finally, test the duplicate cname error response
	duplicateFoo := plugin.NewEntry("foo#bar")
	group.entries = append(group.entries, &duplicateFoo)
	expectedErr := plugin.DuplicateCNameErr{
		ParentID:                        plugin.ID(group),
		FirstChildName:                  foo.Name(),
		FirstChildSlashReplacementChar:  '#',
		SecondChildName:                 duplicateFoo.Name(),
		SecondChildSlashReplacementChar: '#',
		CName:                           "foo#bar",
	}
	runTestCase(
		group,
		testcase{[]string{"foo#bar"}, "", duplicateCNameResponse(expectedErr)},
	)
}

type mockRoot struct {
	plugin.EntryBase
	mock.Mock
}

func (m *mockRoot) Init() error {
	return nil
}

func (m *mockRoot) List(ctx context.Context) ([]plugin.Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}

func getRequest(path string) *http.Request {
	return &http.Request{URL: &url.URL{RawQuery: url.Values{"path": []string{path}}.Encode()}}
}

func (suite *HelpersTestSuite) TestGetEntryFromPath() {
	reg := plugin.NewRegistry()
	plug := &mockRoot{EntryBase: plugin.NewEntry("mine")}
	plug.SetTestID("/mine")
	suite.NoError(reg.RegisterPlugin(plug))
	ctx := context.WithValue(context.Background(), pluginRegistryKey, reg)

	mountpoint := "/mountpoint"
	ctx = context.WithValue(ctx, mountpointKey, mountpoint)

	_, _, err := getEntryFromRequest(ctx, getRequest("relative"))
	suite.Error(relativePathResponse("relative"), err)

	// TODO: Add tests for non-Wash entries (i.e. for apifs)

	entry, path, err := getEntryFromRequest(ctx, getRequest(mountpoint))
	if suite.Nil(err) {
		suite.Equal(mountpoint, path)
		suite.Equal(reg.Name(), plugin.Name(entry))
	}

	entry, path, err = getEntryFromRequest(ctx, getRequest(mountpoint+"/"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/", path)
		suite.Equal(reg.Name(), plugin.Name(entry))
	}

	entry, path, err = getEntryFromRequest(ctx, getRequest(mountpoint+"/mine"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/mine", path)
		suite.Equal(plug.Name(), plugin.Name(entry))
	}

	_, _, err = getEntryFromRequest(ctx, getRequest(mountpoint+"/yours"))
	suite.Error(err)

	file := plugin.NewEntry("a file")
	file.SetTestID("/mine/a file")
	plug.On("List", mock.Anything).Return([]plugin.Entry{&file}, nil)

	entry, path, err = getEntryFromRequest(ctx, getRequest(mountpoint+"/mine/a file"))
	if suite.Nil(err) {
		suite.Equal(mountpoint+"/mine/a file", path)
		suite.Equal(file.Name(), plugin.Name(entry))
	}
	plug.AssertExpectations(suite.T())

	plug.On("List", mock.Anything).Return([]plugin.Entry{&file}, nil)
	_, _, err = getEntryFromRequest(ctx, getRequest(mountpoint+"/mine/a dir"))
	suite.Error(err)
	plug.AssertExpectations(suite.T())
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
