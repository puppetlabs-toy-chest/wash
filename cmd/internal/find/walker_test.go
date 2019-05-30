package find

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/stretchr/testify/mock"
	"github.com/puppetlabs/wash/api/types"
	"github.com/puppetlabs/wash/cmd/internal/cmdtest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/plugin"
)

type WalkerTestSuite struct {
	*cmdtest.Suite
	walker *walker
}

func (s *WalkerTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.walker = newWalker(
		parser.Result{
			Options: types.NewOptions(),
			Predicate: func(e types.Entry) bool {
				return true
			},
		},
		s.Suite.Client,
	)
}

func (s *WalkerTestSuite) TearDownTest() {
	s.Suite.TearDownTest()
	s.walker = nil
	primary.Parser.SetPrimaries = make(map[*primary.Primary]bool)
}

func (s *WalkerTestSuite) TestWalk_InfoErrors()() {
	err := fmt.Errorf("failed to get the info")
	s.Client.On("Info", ".").Return(apitypes.Entry{}, err)
	s.False(s.walker.Walk("."))
	s.Regexp(err.Error(), s.Stderr())
}

func (s *WalkerTestSuite) TestWalk_HappyCase() {
	s.setupMocksForWalk()
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

func (s *WalkerTestSuite) TestWalk_MaxdepthSet() {
	s.setupMocksForWalk()
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
	s.setupMocksForWalk()
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
	s.setupMocksForWalk()
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
	s.setupMocksForWalk()
	err := fmt.Errorf("failed to list")
	s.mockList("./foo", true, nil, err)
	s.False(s.walker.Walk("."))
	s.assertPrintedTree(
		".",
		"./foo",
	)
	s.Regexp("children.*./foo.*" + err.Error(), s.Stderr())
}

func (s *WalkerTestSuite) TestWalk_VisitErrors() {
	s.setupMocksForWalk()
	s.walker.opts.Fullmeta = true
	primary.Parser.SetPrimaries[primary.Meta] = true

	err := fmt.Errorf("failed to fetch metadata")
	s.Client.On("Metadata", mock.Anything).Return(map[string]interface{}{}, err)

	s.False(s.walker.Walk("."))
	s.assertPrintedTree()
	
	// Also test the behavior when depth is set since visit is called
	// on a different code-path
	s.walker.opts.Depth = true
	s.setupMocksForWalk()
	s.False(s.walker.Walk("."))
	s.assertPrintedTree()
}

func (s *WalkerTestSuite) TestVisit_MindepthSet() {
	s.walker.opts.Mindepth = 1
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
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
	s.walker.p = func(entry types.Entry) bool {
		return s.Equal(fullMeta, entry.Metadata)
	}

	e := newMockEntryForVisit()
	s.Client.On("Metadata", e.Path).Return(fullMeta, nil).Once()

	s.walker.visit(e, 0)
	s.Client.AssertCalled(s.T(), "Metadata", e.Path)
	// Ensure that the entry was printed to stdout. This is only true if
	// e.Metadata is set to fullMeta (based on our predicate)
	s.assertPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_PrintsSatisfyingEntry() {
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
	s.assertPrintedEntry(e)
}

func (s *WalkerTestSuite) TestVisit_DoesNotPrintUnsatisfyingEntry() {
	s.walker.p = func(e types.Entry) bool {
		return false
	}
	e := newMockEntryForVisit()
	s.True(s.walker.visit(e, 0))
	s.assertNotPrintedEntry(e)
}

func (s *WalkerTestSuite) setupMocksForWalk() {
	toEntry := func(path string, isParent bool) apitypes.Entry {
		e := apitypes.Entry{
			Path: s.toAbsPath(path),
			CName: filepath.Base(path),
		}
		if isParent {
			e.Actions = []string{"list"}
		}
		return e
	}

	// Mock-out "Info" for the root
	s.Client.On("Info", ".").Return(toEntry(".", true), nil).Once()

	// Mock out "List" for each entry in the tree
	tree := map[string][]apitypes.Entry{
		".": []apitypes.Entry{toEntry("./foo", true)},
		"./foo": []apitypes.Entry{
			toEntry("./foo/bar", true),
			toEntry("./foo/baz", false),
		},
		"./foo/bar": []apitypes.Entry{
			toEntry("./foo/bar/1", false),
			toEntry("./foo/bar/2", false),
		},
	}
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

