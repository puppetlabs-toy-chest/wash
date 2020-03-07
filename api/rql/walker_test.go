package rql

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type WalkerTestSuite struct {
	suite.Suite
	walker *walkerImpl
}

func (s *WalkerTestSuite) SetupTest() {
	plugin.SetTestCache(datastore.NewMemCache())
	s.walker = newWalker(
		&mockQuery{
			EntryP: func(e Entry) bool {
				return true
			},
		},
		NewOptions(),
	).(*walkerImpl)
}

func (s *WalkerTestSuite) TearDownTest() {
	plugin.UnsetTestCache()
	s.walker = nil
}

func (s *WalkerTestSuite) TestWalk_SchemaErrors() {
	e := newMockPluginEntry(".")
	expectedErr := fmt.Errorf("failed to get the schema")
	e.On("SchemaGraph").Return((*linkedhashmap.Map)(nil), expectedErr)
	_, err := s.walker.Walk(context.Background(), e)
	s.Regexp(expectedErr.Error(), err)
}

/*
TODO: Enable this test once the SchemaRequired optimization is in-place
func (s *WalkerTestSuite) TestWalk_SchemaRequired_UnknownSchema() {
	s.walker.p.RequireSchema()
	s.setupDefaultMocksForWalk()
	s.True(s.walker.Walk("."))
	s.Empty(s.Stdout())
	s.Empty(s.Stderr())
	s.Client.AssertNotCalled(s.T(), "List", ".")
}
*/

// NOTE: Remember that the start path is eliminated to ensure the
// correctness of the path primary. Thus, "" represents the starting
// entry in the results.

func (s *WalkerTestSuite) TestWalk_HappyCase() {
	tree := s.setupDefaultMocksForWalk()
	entries := s.mustWalk(context.Background(), tree["."])
	s.assertEntries(
		[]string{
			"foo",
			"foo/bar",
			"foo/bar/1",
			"foo/bar/2",
			"foo/baz",
		},
		entries,
		nil,
	)
}

func (s *WalkerTestSuite) TestWalk_WithSchema_HappyCase() {
	// Set-up the mocks
	fileSchema := func(label string) plugin.EntrySchema {
		schema := plugin.NewEntrySchema(nil, label)
		return (*schema)
	}
	dirSchema := func(label string, children []string) plugin.EntrySchema {
		schema := plugin.NewEntrySchema(nil, label)
		schema.Children = children
		return (*schema)
	}
	schemaGraph := linkedhashmap.New()
	// Since the ID begins with ".", the "pluginName" is "." which is why
	// these typeIDs are namespaced as ".::"
	schemaGraph.Put(".::root", dirSchema(".", []string{".::foo", ".::bar", ".::baz"}))
	schemaGraph.Put(".::foo", dirSchema("foo", []string{".::file"}))
	schemaGraph.Put(".::bar", dirSchema("bar", []string{".::no_file"}))
	schemaGraph.Put(".::baz", dirSchema("baz", []string{".::file"}))
	schemaGraph.Put(".::file", fileSchema("file"))
	schemaGraph.Put(".::no_file", fileSchema("no_file"))
	tree := s.setupMocksForWalk(schemaGraph, map[string]*mockPluginEntry{
		".":       s.toPluginEntry(".", true, "root"),
		"./foo":   s.toPluginEntry("./foo", true, "foo"),
		"./bar":   s.toPluginEntry("./bar", true, "bar"),
		"./baz":   s.toPluginEntry("./baz", true, "baz"),
		"./foo/1": s.toPluginEntry("./foo/1", false, "file"),
		"./bar/1": s.toPluginEntry("./bar/1", false, "no_file"),
		"./bar/2": s.toPluginEntry("./bar/2", false, "no_file"),
		"./baz/1": s.toPluginEntry("./baz/1", false, "file"),
		"./baz/2": s.toPluginEntry("./baz/2", false, "file"),
	})

	s.walker.q.(*mockQuery).EntrySchemaP = func(s *EntrySchema) bool {
		// Print only "foo" or "file"s. Ignore everything else.
		rx := regexp.MustCompile(`(^foo)|(/file$)`)
		return rx.MatchString(s.Path())
	}

	entries := s.mustWalk(context.Background(), tree["."])
	s.assertEntries(
		[]string{
			"baz/1",
			"baz/2",
			"foo",
			"foo/1",
		},
		entries,
		nil,
	)
}

func (s *WalkerTestSuite) TestWalk_MaxdepthSet() {
	tree := s.setupDefaultMocksForWalk()
	s.walker.opts.Maxdepth = 2
	entries := s.mustWalk(context.Background(), tree["."])
	s.assertEntries(
		[]string{
			"foo",
			"foo/bar",
			"foo/baz",
		},
		entries,
		nil,
	)
}

func (s *WalkerTestSuite) TestWalk_MaxdepthAndMindepthSet() {
	tree := s.setupDefaultMocksForWalk()
	s.walker.opts.Mindepth = 1
	s.walker.opts.Maxdepth = 2
	entries := s.mustWalk(context.Background(), tree["."])
	s.assertEntries(
		[]string{
			"foo",
			"foo/bar",
			"foo/baz",
		},
		entries,
		nil,
	)
}

func (s *WalkerTestSuite) TestWalk_ListErrors() {
	tree := s.setupDefaultMocksForWalk()
	expectedErr := fmt.Errorf("failed to list")
	s.mockList(tree["./foo"], true, nil, expectedErr)
	_, err := s.walker.Walk(context.Background(), tree["."])
	s.Regexp("children.*foo.*"+expectedErr.Error(), err)
}

func (s *WalkerTestSuite) TestWalk_VisitErrors() {
	tree := s.setupDefaultMocksForWalk()
	s.walker.opts.Fullmeta = true

	expectedErr := fmt.Errorf("failed to fetch metadata")
	tree["./foo"].On("Metadata", mock.Anything).Return(plugin.JSONObject{}, expectedErr)

	_, err := s.walker.Walk(context.Background(), tree["."])
	s.Regexp(`full.*metadata.*failed.*metadata`, err)
}

func (s *WalkerTestSuite) TestVisit_MindepthSet() {
	s.walker.opts.Mindepth = 1
	e := newMockEntryForVisit()
	s.False(s.mustVisit(context.Background(), &e, 0))
}

func (s *WalkerTestSuite) TestVisit_UnsatisfyingSchema_DoesNotVisit() {
	e := newMockEntryForVisit()
	e.Schema = &EntrySchema{}
	s.walker.q.(*mockQuery).EntrySchemaP = func(_ *EntrySchema) bool {
		return false
	}
	s.False(s.mustVisit(context.Background(), &e, 0))
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_FailsToFetchFullMetadata() {
	s.walker.opts.Fullmeta = true

	e := newMockEntryForVisit()
	pluginEntry := e.pluginEntry.(*mockPluginEntry)
	expectedErr := fmt.Errorf("failed to fetch metadata")
	pluginEntry.On("Metadata", mock.Anything).Return(plugin.JSONObject{}, expectedErr)

	_, err := s.walker.visit(context.Background(), &e, 0)
	s.Regexp(expectedErr.Error(), err)
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_FetchesFullMetadata() {
	s.walker.opts.Fullmeta = true

	fullMeta := plugin.JSONObject{"foo": "bar"}
	s.walker.q.(*mockQuery).EntryP = func(entry Entry) bool {
		return s.Equal(fullMeta, entry.Metadata)
	}

	e := newMockEntryForVisit()
	pluginEntry := e.pluginEntry.(*mockPluginEntry)
	pluginEntry.On("Metadata", mock.Anything).Return(fullMeta, nil).Once()

	s.True(s.mustVisit(context.Background(), &e, 0))
	s.Equal(fullMeta, e.Metadata)
	pluginEntry.AssertCalled(s.T(), "Metadata", mock.Anything)
}

func (s *WalkerTestSuite) TestVisit_ReturnsTrueForSatisfyingEntry() {
	e := newMockEntryForVisit()
	s.True(s.mustVisit(context.Background(), &e, 0))
}

func (s *WalkerTestSuite) TestVisit_ReturnsFalseForUnsatisfyingEntry() {
	s.walker.q.(*mockQuery).EntryP = func(Entry) bool {
		return false
	}
	e := newMockEntryForVisit()
	s.False(s.mustVisit(context.Background(), &e, 0))
}

func (s *WalkerTestSuite) setupDefaultMocksForWalk() map[string]*mockPluginEntry {
	return s.setupMocksForWalk(nil, map[string]*mockPluginEntry{
		".":           s.toPluginEntry(".", true, ""),
		"./foo":       s.toPluginEntry("./foo", true, ""),
		"./foo/bar":   s.toPluginEntry("./foo/bar", true, ""),
		"./foo/baz":   s.toPluginEntry("./foo/baz", false, ""),
		"./foo/bar/1": s.toPluginEntry("./foo/bar/1", false, ""),
		"./foo/bar/2": s.toPluginEntry("./foo/bar/2", false, ""),
	})
}

func (s *WalkerTestSuite) setupMocksForWalk(schemaGraph *linkedhashmap.Map, tree map[string]*mockPluginEntry) map[string]*mockPluginEntry {
	// Mock-out "SchemaGrph" for the root
	tree["."].On("SchemaGraph").Return(schemaGraph, nil).Once()
	// Construct the map of <id> => <children>
	childrenMap := make(map[string][]plugin.Entry)
	for id, entry := range tree {
		if !entry.isNotParent {
			// Entry is a parent so make sure that something's in the children
			// map in case the entry doesn't have any children
			_, ok := childrenMap[id]
			if !ok {
				childrenMap[id] = []plugin.Entry{}
			}
			if !strings.Contains(id, "/") {
				// Root has no parent so nothing more needs to be done
				continue
			}
		}
		segments := strings.Split(id, "/")
		segments = segments[0 : len(segments)-1]
		parent := strings.Join(segments, "/")
		childrenMap[parent] = append(childrenMap[parent], entry)
	}
	// Mock out "List" for each parent entry in the tree
	for id, children := range childrenMap {
		s.mockList(tree[id], false, children, nil)
	}
	return tree
}

func (s *WalkerTestSuite) mockList(entry *mockPluginEntry, previouslyMocked bool, children []plugin.Entry, err error) {
	if previouslyMocked {
		// Erase the existing mocks by invoking them
		_, _ = entry.List(context.Background())
	}
	entry.On("List", mock.Anything).Return(children, err).Once()
}

func (s *WalkerTestSuite) toPluginEntry(path string, isParent bool, typeID string) *mockPluginEntry {
	name := filepath.Base(path)
	pluginEntry := newMockPluginEntry(name)
	pluginEntry.isNotParent = !isParent
	pluginEntry.rawTypeID = typeID
	pluginEntry.SetTestID(path)
	return pluginEntry
}

func (s *WalkerTestSuite) mustWalk(ctx context.Context, start plugin.Entry) []Entry {
	entries, err := s.walker.Walk(ctx, start)
	if err != nil {
		s.FailNow(fmt.Sprintf("expected Walk to not error but got %v", err))
	}
	return entries
}

func (s *WalkerTestSuite) mustVisit(ctx context.Context, e *Entry, depth int) bool {
	includeEntry, err := s.walker.visit(ctx, e, depth)
	if err != nil {
		s.FailNow(fmt.Sprintf("expected visit to not error but got %v", err))
	}
	return includeEntry
}

func (s *WalkerTestSuite) assertEntries(expectedPaths []string, actualEntries []Entry, additionalAssertions func(Entry)) {
	var actualPaths []string
	for _, entry := range actualEntries {
		actualPaths = append(actualPaths, entry.Path)
	}
	s.Equal(expectedPaths, actualPaths)
	if additionalAssertions != nil {
		for _, entry := range actualEntries {
			additionalAssertions(entry)
		}
	}
}

func newMockEntryForVisit() Entry {
	e := newEntry(nil, newMockPluginEntry("foo"))
	return e
}

func TestWalker(t *testing.T) {
	suite.Run(t, new(WalkerTestSuite))
}

type mockQuery struct {
	EntryP       func(Entry) bool
	EntrySchemaP func(*EntrySchema) bool
}

func (p *mockQuery) Marshal() interface{} {
	return nil
}

func (p *mockQuery) Unmarshal(input interface{}) error {
	return nil
}

func (p *mockQuery) EvalEntry(e Entry) bool {
	return p.EntryP(e)
}

func (p *mockQuery) EvalEntrySchema(s *EntrySchema) bool {
	return p.EntrySchemaP(s)
}

var _ = EntryPredicate(&mockQuery{})
var _ = EntrySchemaPredicate(&mockQuery{})

type mockPluginEntry struct {
	plugin.EntryBase
	mock.Mock
	rawTypeID   string
	isNotParent bool
}

func newMockPluginEntry(name string) *mockPluginEntry {
	e := &mockPluginEntry{
		EntryBase: plugin.NewEntry(name),
	}
	e.DisableDefaultCaching()
	return e
}

func (m *mockPluginEntry) Schema() *plugin.EntrySchema {
	return nil
}

func (m *mockPluginEntry) ChildSchemas() []*plugin.EntrySchema {
	return nil
}

func (m *mockPluginEntry) List(ctx context.Context) ([]plugin.Entry, error) {
	if m.isNotParent {
		return nil, fmt.Errorf("List called on a non-parent mock plugin entry")
	}
	args := m.Called(ctx)
	return args.Get(0).([]plugin.Entry), args.Error(1)
}

func (m *mockPluginEntry) Metadata(ctx context.Context) (plugin.JSONObject, error) {
	args := m.Called(ctx)
	return args.Get(0).(plugin.JSONObject), args.Error(1)
}

// Mock the external plugin interface so that we have the ability
// to mock schemas, type IDs, supported methods, etc.
//
// TODO: Refactor plugin.externalPlugin to separate out stuff that
// external plugins need vs stuff that lets people mock entry stuff.

func (m *mockPluginEntry) MethodSignature(method string) plugin.MethodSignature {
	if method == plugin.ListAction().Name && m.isNotParent {
		return plugin.UnsupportedSignature
	}
	return plugin.DefaultSignature
}

func (m *mockPluginEntry) SchemaGraph() (*linkedhashmap.Map, error) {
	args := m.Called()
	return args.Get(0).(*linkedhashmap.Map), args.Error(1)
}

func (m *mockPluginEntry) RawTypeID() string {
	return m.rawTypeID
}

func (m *mockPluginEntry) BlockRead(ctx context.Context, size int64, offset int64) ([]byte, error) {
	panic("BlockRead should never be called")
}

var _ = plugin.Parent(&mockPluginEntry{})
