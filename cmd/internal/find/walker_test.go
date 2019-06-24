package find

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/cmd/internal/cmdtest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type WalkerTestSuite struct {
	*cmdtest.Suite
	walker *walkerImpl
}

func (s *WalkerTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.walker = newWalker(
		parser.Result{
			Options: types.NewOptions(),
			Predicate: types.ToEntryP(func(e types.Entry) bool {
				return true
			}),
		},
		s.Suite.Client,
	).(*walkerImpl)
}

func (s *WalkerTestSuite) TearDownTest() {
	s.Suite.TearDownTest()
	s.walker = nil
	primary.Parser.SetPrimaries = make(map[*primary.Primary]bool)
}

func (s *WalkerTestSuite) TestWalk_InfoErrors() {
	err := fmt.Errorf("failed to get the info")
	s.Client.On("Info", ".").Return(apitypes.Entry{}, err)
	s.False(s.walker.Walk("."))
	s.Regexp(err.Error(), s.Stderr())
}

func (s *WalkerTestSuite) TestWalk_SchemaErrors() {
	s.Client.On("Info", ".").Return(apitypes.Entry{}, nil)
	err := fmt.Errorf("failed to get the schema")
	s.Client.On("Schema", ".").Return((*apitypes.EntrySchema)(nil), err)
	s.False(s.walker.Walk("."))
	s.Regexp(err.Error(), s.Stderr())
}

func (s *WalkerTestSuite) TestWalk_HappyCase() {
	s.setupDefaultMocksForWalk()
	s.True(s.walker.Walk("."))
	s.assertPrintedTree(
		".",
		"./foo",
		"./foo/bar",
		"./foo/bar/1",
		"./foo/bar/2",
		"./foo/baz",
	)
}

func (s *WalkerTestSuite) TestWalk_WithSchema_HappyCase() {
	// Set-up the mocks
	fileSchema := func(typeID string) *apitypes.EntrySchema {
		return (&apitypes.EntrySchema{}).SetTypeID(typeID)
	}
	dirSchema := func(typeID string, children ...*apitypes.EntrySchema) *apitypes.EntrySchema {
		return (&apitypes.EntrySchema{}).SetTypeID(typeID).SetChildren(children)
	}
	schema := dirSchema(
		".",
		dirSchema("foo", fileSchema("file")),
		dirSchema("bar", fileSchema("no_file")),
		dirSchema("baz", fileSchema("file")),
	)
	s.setupMocksForWalk(schema, map[string][]apitypes.Entry{
		".": []apitypes.Entry{
			s.toEntry("./foo", true, "foo"),
			s.toEntry("./bar", true, "bar"),
			s.toEntry("./baz", true, "baz"),
		},
		"./foo": []apitypes.Entry{
			s.toEntry("./foo/1", true, "file"),
		},
		"./bar": []apitypes.Entry{
			s.toEntry("./bar/1", false, "no_file"),
			s.toEntry("./bar/2", false, "no_file"),
		},
		"./baz": []apitypes.Entry{
			s.toEntry("./baz/1", false, "file"),
			s.toEntry("./baz/2", false, "file"),
		},
	})

	s.walker.p.SchemaP = func(s *types.EntrySchema) bool {
		// Print only "foo" or "file"s. Ignore everything else.
		return s.TypeID() == "foo" || s.TypeID() == "file"
	}

	s.True(s.walker.Walk("."))
	s.assertPrintedTree(
		"./foo",
		"./foo/1",
		"./baz/1",
		"./baz/2",
	)
}

func (s *WalkerTestSuite) TestWalk_MaxdepthSet() {
	s.setupDefaultMocksForWalk()
	s.walker.opts.Maxdepth = 2
	s.walker.opts.MarkAsSet(types.MaxdepthFlag)
	s.True(s.walker.Walk("."))
	s.assertPrintedTree(
		".",
		"./foo",
		"./foo/bar",
		"./foo/baz",
	)
}

func (s *WalkerTestSuite) TestWalk_MaxdepthAndMindepthSet() {
	s.setupDefaultMocksForWalk()
	s.walker.opts.Mindepth = 1
	s.walker.opts.Maxdepth = 2
	s.True(s.walker.Walk("."))
	s.assertPrintedTree(
		"./foo",
		"./foo/bar",
		"./foo/baz",
	)
}

func (s *WalkerTestSuite) TestWalk_DepthSet() {
	s.setupDefaultMocksForWalk()
	s.walker.opts.Depth = true
	s.True(s.walker.Walk("."))
	s.assertPrintedTree(
		"./foo/bar/1",
		"./foo/bar/2",
		"./foo/bar",
		"./foo/baz",
		"./foo",
		".",
	)
}

func (s *WalkerTestSuite) TestWalk_ListErrors() {
	s.setupDefaultMocksForWalk()
	err := fmt.Errorf("failed to list")
	s.mockList("./foo", true, nil, err)
	s.False(s.walker.Walk("."))
	s.assertPrintedTree(
		".",
		"./foo",
	)
	s.Regexp("children.*./foo.*"+err.Error(), s.Stderr())
}

func (s *WalkerTestSuite) TestWalk_VisitErrors() {
	s.setupDefaultMocksForWalk()
	s.walker.opts.Fullmeta = true
	primary.Parser.SetPrimaries[primary.Meta] = true

	err := fmt.Errorf("failed to fetch metadata")
	s.Client.On("Metadata", mock.Anything).Return(map[string]interface{}{}, err)

	s.False(s.walker.Walk("."))
	s.assertPrintedTree()

	// Also test the behavior when depth is set since visit is called
	// on a different code-path
	s.walker.opts.Depth = true
	s.setupDefaultMocksForWalk()
	s.False(s.walker.Walk("."))
	s.assertPrintedTree()
}

func (s *WalkerTestSuite) TestVisit_MindepthSet() {
	s.walker.opts.Mindepth = 1
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_NilSchema_DoesNotVisit() {
	e := newMockEntryForVisit()
	e.SetSchema(nil)
	s.walker.visit(e, 0)
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_UnsatisfyingSchema_DoesNotVisit() {
	e := newMockEntryForVisit()
	e.SetSchema(&types.EntrySchema{})
	s.walker.p.SchemaP = func(_ *types.EntrySchema) bool {
		return false
	}
	s.walker.visit(e, 0)
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_MetaPrimaryUnset_DoesNotFetchFullMetadata() {
	s.walker.opts.Fullmeta = true
	e := newMockEntryForVisit()
	s.walker.visit(e, 0)
	// Ensure that the entry's full metadata was not fetched
	s.Client.AssertNotCalled(s.T(), "Metadata")
	// Ensure that the entry was still printed to avoid false positives due to
	// e.g. forgetting something in the setup
	s.assertPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_MetaPrimarySet_FailsToFetchFullMetadata() {
	s.walker.opts.Fullmeta = true
	primary.Parser.SetPrimaries[primary.Meta] = true

	e := newMockEntryForVisit()
	err := fmt.Errorf("failed to fetch metadata")
	s.Client.On("Metadata", e.Path).Return(map[string]interface{}{}, err)

	s.False(s.walker.visit(e, 0))
	s.Regexp(err.Error(), s.Stderr())
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_MetaPrimarySet_FetchesFullMetadata() {
	s.walker.opts.Fullmeta = true
	primary.Parser.SetPrimaries[primary.Meta] = true

	fullMeta := plugin.JSONObject{"foo": "bar"}
	s.walker.p = types.ToEntryP(func(entry types.Entry) bool {
		return s.Equal(fullMeta, entry.Metadata)
	})

	e := newMockEntryForVisit()
	s.Client.On("Metadata", e.Path).Return(fullMeta, nil).Once()

	s.walker.visit(e, 0)
	s.Client.AssertCalled(s.T(), "Metadata", e.Path)
	// Ensure that the entry was printed to stdout. This is only true if
	// e.Metadata is set to fullMeta (based on our predicate)
	s.assertPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_FullmetaSet_MetaPrimarySet_UnsatisfyingSchema_DoesNotFetchFullMetadata() {
	// This test is a sanity check since O(N) metadata requests can be expensive
	s.walker.opts.Fullmeta = true
	primary.Parser.SetPrimaries[primary.Meta] = true
	e := newMockEntryForVisit()
	e.SetSchema(nil)
	s.walker.visit(e, 0)
	s.Client.AssertNotCalled(s.T(), "Metadata")
}

func (s *WalkerTestSuite) TestVisit_PrintsSatisfyingEntry() {
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
	s.assertPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_DoesNotPrintUnsatisfyingEntry() {
	s.walker.p = types.ToEntryP(func(e types.Entry) bool {
		return false
	})
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) setupDefaultMocksForWalk() {
	s.setupMocksForWalk(nil, map[string][]apitypes.Entry{
		".": []apitypes.Entry{s.toEntry("./foo", true, "")},
		"./foo": []apitypes.Entry{
			s.toEntry("./foo/bar", true, ""),
			s.toEntry("./foo/baz", false, ""),
		},
		"./foo/bar": []apitypes.Entry{
			s.toEntry("./foo/bar/1", false, ""),
			s.toEntry("./foo/bar/2", false, ""),
		},
	})
}

func (s *WalkerTestSuite) setupMocksForWalk(schema *apitypes.EntrySchema, tree map[string][]apitypes.Entry) {
	// Mock-out "Info" + "Schema" for the root
	s.Client.On("Info", ".").Return(s.toEntry(".", true, "."), nil).Once()
	s.Client.On("Schema", ".").Return(schema, nil).Once()
	// Mock out "List" for each entry in the tree
	for dir, children := range tree {
		s.mockList(dir, false, children, nil)
	}
}

func (s *WalkerTestSuite) mockList(path string, previouslyMocked bool, children []apitypes.Entry, err error) {
	absPath := s.toAbsPath(path)
	if previouslyMocked {
		// Erase the existing mocks by invoking them
		_, _ = s.Client.List(path)
		_, _ = s.Client.List(absPath)
	}
	s.Client.On("List", path).Return(children, err).Once()
	s.Client.On("List", absPath).Return(children, err).Once()
}

func (s *WalkerTestSuite) toAbsPath(path string) string {
	if path == "." {
		return "/"
	}
	return strings.TrimPrefix(path, ".")
}

func (s *WalkerTestSuite) toEntry(path string, isParent bool, typeID string) apitypes.Entry {
	e := apitypes.Entry{
		Path:   s.toAbsPath(path),
		CName:  filepath.Base(path),
		TypeID: typeID,
	}
	if isParent {
		e.Actions = []string{"list"}
	}
	return e
}

func (s *WalkerTestSuite) assertPrintedEntry(e types.Entry) {
	s.Regexp(e.NormalizedPath, s.Stdout())
}

func (s *WalkerTestSuite) assertNotPrintedEntry(e types.Entry) {
	s.NotRegexp(e.NormalizedPath, s.Stdout())
}

func (s *WalkerTestSuite) assertPrintedTree(paths ...string) {
	expectedStdout := strings.Join(paths, "\n")
	if expectedStdout != "" {
		expectedStdout += "\n"
	}
	s.Equal(expectedStdout, s.Stdout())
}

func newMockEntryForVisit() types.Entry {
	e := types.Entry{}
	e.Path = "/foo"
	e.NormalizedPath = "./foo"
	return e
}

func TestWalker(t *testing.T) {
	s := new(WalkerTestSuite)
	s.Suite = new(cmdtest.Suite)
	suite.Run(t, s)
}
