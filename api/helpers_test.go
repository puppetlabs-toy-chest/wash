package api

import (
	"context"
	"os"
	"testing"
	"time"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/plugin"
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

func (suite *HelpersTestSuite) TestToAPIAttrAtime() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetAtime(time.Now())
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Atime) {
		suite.Equal(a.Atime(), *apiAttr.Atime)
	}
}

func (suite *HelpersTestSuite) TestToAPIAttrMtime() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetMtime(time.Now())
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Mtime) {
		suite.Equal(a.Mtime(), *apiAttr.Mtime)
	}
}

func (suite *HelpersTestSuite) TestToAPIAttrCtime() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetCtime(time.Now())
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Ctime) {
		suite.Equal(a.Ctime(), *apiAttr.Ctime)
	}
}

func (suite *HelpersTestSuite) TestToAPIAttrMode() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetMode(0777)
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Mode) {
		suite.Equal(a.Mode(), *apiAttr.Mode)
	}
}

func (suite *HelpersTestSuite) TestToAPIAttrSize() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetSize(10)
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Size) {
		suite.Equal(a.Size(), *apiAttr.Size)
	}
}

func (suite *HelpersTestSuite) TestToAPIAttrMultipleAttr() {
	a := plugin.EntryAttributes{}
	suite.Equal(apitypes.EntryAttributes{}, toAPIAttr(a))

	a.SetMode(os.FileMode(0777))
	a.SetSize(10)
	apiAttr := toAPIAttr(a)
	if suite.NotNil(apiAttr.Mode) && suite.NotNil(apiAttr.Size) {
		suite.Equal(a.Mode(), *apiAttr.Mode)
		suite.Equal(a.Size(), *apiAttr.Size)
	}
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
		ParentPath:                      plugin.Path(group),
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

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
