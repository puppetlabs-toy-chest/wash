package api

import (
	"context"
	"testing"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
)

type mockGroup struct {
	plugin.EntryBase
	entries []plugin.Entry
}

func (g *mockGroup) LS(context.Context) ([]plugin.Entry, error) {
	return g.entries, nil
}

func TestFindEntry(t *testing.T) {
	type testcase struct {
		segments      []string
		expectedEntry string
		expectedErr   error
	}
	runTestCase := func(grp plugin.Group, c testcase) {
		got, err := findEntry(context.Background(), grp, c.segments)
		if c.expectedEntry != "" && assert.NotNil(t, got) {
			assert.Equal(t, c.expectedEntry, got.Name())
		} else {
			assert.Nil(t, got)
		}
		if c.expectedErr == nil {
			assert.Nil(t, err)
		} else {
			assert.Equal(t, c.expectedErr, err)
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
