package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/datastore"
	"github.com/stretchr/testify/assert"
)

type mockParent struct {
	EntryBase
	entries []Entry
}

func (g *mockParent) List(context.Context) ([]Entry, error) {
	return g.entries, nil
}

func (g *mockParent) ChildSchemas() []*EntrySchema {
	return nil
}

func (g *mockParent) Schema() *EntrySchema {
	return nil
}

type mockEntry struct {
	EntryBase
}

func newMockEntry(name string) *mockEntry {
	return &mockEntry{
		EntryBase: NewEntry(name),
	}
}

func (e *mockEntry) Schema() *EntrySchema {
	return nil
}

func TestFindEntry(t *testing.T) {
	SetTestCache(datastore.NewMemCache())
	defer UnsetTestCache()

	type testcase struct {
		segments      []string
		expectedEntry string
		expectedErr   error
	}
	runTestCase := func(parent Parent, c testcase) {
		got, err := FindEntry(context.Background(), parent, c.segments)
		if c.expectedEntry != "" && assert.NotNil(t, got) {
			assert.Equal(t, c.expectedEntry, CName(got))
		} else {
			assert.Nil(t, got)
		}
		if c.expectedErr == nil {
			assert.Nil(t, err)
		} else {
			assert.Equal(t, c.expectedErr, err)
		}
	}

	foo := newMockEntry("foo/bar")
	parent := &mockParent{NewEntry("root"), []Entry{foo}}
	parent.SetTestID("/root")
	parent.DisableDefaultCaching()
	for _, c := range []testcase{
		{[]string{"not found"}, "", fmt.Errorf("The not found entry does not exist")},
		{[]string{"foo#bar"}, "foo#bar", nil},
		{[]string{"foo#bar", "bar"}, "", fmt.Errorf("The entry foo#bar is not a parent")},
	} {
		runTestCase(parent, c)
	}

	baz := newMockEntry("baz")
	nestedParent := &mockParent{NewEntry("bar"), []Entry{baz}}
	nestedParent.DisableDefaultCaching()
	parent.entries = append(parent.entries, nestedParent)
	for _, c := range []testcase{
		{[]string{"bar"}, "bar", nil},
		{[]string{"bar", "foo"}, "", fmt.Errorf("The foo entry does not exist in the bar parent")},
		{[]string{"bar", "baz"}, "baz", nil},
	} {
		runTestCase(parent, c)
	}

	// Finally, test the duplicate cname error response
	duplicateFoo := newMockEntry("foo#bar")
	parent.entries = append(parent.entries, duplicateFoo)
	expectedErr := DuplicateCNameErr{
		ParentID:                 ID(parent),
		FirstChildName:           foo.Name(),
		FirstChildSlashReplacer:  '#',
		SecondChildName:          duplicateFoo.Name(),
		SecondChildSlashReplacer: '#',
		CName:                    "foo#bar",
	}
	runTestCase(
		parent,
		testcase{[]string{"foo#bar"}, "", expectedErr},
	)
}
