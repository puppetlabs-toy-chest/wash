package api

import (
	"context"

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

func (suite *HelpersTestSuite) TestFindEntry() {
	type testcase struct {
		segments      []string
		expectedEntry string
		expectedErr   error
	}
	runTestCase := func(grp plugin.Group, c testcase) {
		got, err := findEntry(context.Background(), grp, c.segments)
		if c.expectedEntry != "" && suite.NotNil(got) {
			suite.Equal(c.expectedEntry, got.Name())
		} else {
			suite.Nil(got)
		}
		if c.expectedErr == nil {
			suite.Nil(err)
		} else {
			suite.Equal(c.expectedErr, err)
		}
	}

	foo := plugin.NewEntry("foo")
	group := &mockGroup{plugin.NewEntry("root"), []plugin.Entry{&foo}}
	for _, c := range []testcase{
		{[]string{"not found"}, "", entryNotFoundResponse("not found", "The not found entry does not exist")},
		{[]string{"foo"}, "foo", nil},
		{[]string{"foo", "bar"}, "", entryNotFoundResponse("foo/bar", "The entry foo is not a group")},
	} {
		runTestCase(group, c)
	}

	baz := plugin.NewEntry("baz")
	group.entries = append(group.entries, &mockGroup{plugin.NewEntry("bar"), []plugin.Entry{&baz}})
	for _, c := range []testcase{
		{[]string{"bar"}, "bar", nil},
		{[]string{"bar", "foo"}, "", entryNotFoundResponse("bar/foo", "The foo entry does not exist in the bar group")},
		{[]string{"bar", "baz"}, "baz", nil},
	} {
		runTestCase(group, c)
	}
}
