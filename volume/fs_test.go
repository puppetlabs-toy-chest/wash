package volume

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// Represents the output of StatCmd(/var/log)
const varLogDepth = 7
const varLogFixture = `
96 1550611510 1550611448 1550611448 41ed /var/log/path
96 1550611510 1550611448 1550611448 41ed /var/log/path/has
96 1550611510 1550611448 1550611448 41ed /var/log/path/has/got
96 1550611510 1550611458 1550611458 41ed /var/log/path/has/got/some
0 1550611458 1550611458 1550611458 81a4 /var/log/path/has/got/some/legs
96 1550611510 1550611453 1550611453 41ed /var/log/path1
0 1550611453 1550611453 1550611453 81a4 /var/log/path1/a file
96 1550611510 1550611441 1550611441 41ed /var/log/path2
64 1550611510 1550611441 1550611441 41ed /var/log/path2/dir
`

type fsTestSuite struct {
	suite.Suite
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (suite *fsTestSuite) SetupTest() {
	// Use a different cache each test because we may re-use some but not all of the same structure.
	plugin.SetTestCache(datastore.NewMemCache())
	suite.ctx, suite.cancelFunc = context.WithCancel(context.Background())
}

func (suite *fsTestSuite) TearDownTest() {
	plugin.UnsetTestCache()
	// Cancelling the context ensures that the tests don't leave any
	// dangling goroutines waiting on a context cancellation. These
	// goroutines are created by ExecCommandImpl
	suite.cancelFunc()
}

func (suite *fsTestSuite) createResult(data string) plugin.ExecCommand {
	cmd := plugin.NewExecCommand(suite.ctx)
	go func() {
		_, err := cmd.Stdout().Write([]byte(data))
		if err != nil {
			msg := fmt.Sprintf("Unexpected error while setting up mocks: %v", err)
			panic(msg)
		}
		cmd.CloseStreamsWithError(nil)
		cmd.SetExitCode(0)
	}()
	return cmd
}

func (suite *fsTestSuite) createExec(fixt string, depth int) *mockExecutor {
	exec := &mockExecutor{EntryBase: plugin.NewEntry("instance")}
	// Used when recording activity.
	exec.SetTestID("/instance")
	cmd := StatCmd("/", depth)
	exec.On("Exec", mock.Anything, cmd[0], cmd[1:], plugin.ExecOptions{Elevate: true}).Return(suite.createResult(fixt), nil)
	return exec
}

func (suite *fsTestSuite) find(parent plugin.Parent, path string) plugin.Entry {
	names := strings.Split(path, "/")
	entry := plugin.Entry(parent)
	for _, name := range names {
		entries, err := plugin.List(suite.ctx, entry.(plugin.Parent))
		if !suite.NoError(err) {
			suite.FailNow("Listing entries failed")
		}
		entries.Range(func(nm string, match plugin.Entry) bool {
			if nm == name {
				entry = match
				return false
			}
			return true
		})
	}
	suite.Equal(names[len(names)-1], plugin.Name(entry))
	return entry
}

func (suite *fsTestSuite) TestFSList() {
	exec := suite.createExec(varLogFixture, varLogDepth)
	fs := NewFS("fs", exec, varLogDepth)
	// ID would normally be set when listing FS within the parent instance.
	fs.SetTestID("/instance/fs")

	entry := suite.find(fs, "var/log").(plugin.Parent)
	entries, err := entry.List(context.Background())
	if !suite.NoError(err) {
		suite.FailNow("Listing entries failed")
	}
	suite.Equal(3, len(entries))

	// Ensure entries are sorted
	sort.Slice(entries, func(i, j int) bool { return plugin.Name(entries[i]) < plugin.Name(entries[j]) })

	suite.Equal("path", plugin.Name(entries[0]))
	suite.Equal("path1", plugin.Name(entries[1]))
	suite.Equal("path2", plugin.Name(entries[2]))
	for _, entry := range entries {
		_, ok := entry.(plugin.Parent)
		if !suite.True(ok) {
			suite.FailNow("Entry was not a Group")
		}
	}

	entries1, err := entries[1].(plugin.Parent).List(context.Background())
	if suite.NoError(err) {
		suite.Equal(1, len(entries1))
		suite.Equal("a file", plugin.Name(entries1[0]))
		_, ok := entries1[0].(plugin.Readable)
		suite.True(ok)
	}

	entries2, err := entries[2].(plugin.Parent).List(context.Background())
	if suite.NoError(err) {
		suite.Equal(1, len(entries2))
		suite.Equal("dir", plugin.Name(entries2[0]))
		_, ok := entries2[0].(plugin.Parent)
		suite.True(ok)
	}
}

func (suite *fsTestSuite) TestFSListTwice() {
	firstFixture := `
96 1550611510 1550611448 1550611448 41ed /var
96 1550611510 1550611448 1550611448 41ed /var/log
96 1550611510 1550611448 1550611448 41ed /var/log/path
`
	secondFixture := `
96 1550611510 1550611448 1550611448 41ed /var/log/path/has
96 1550611510 1550611448 1550611448 41ed /var/log/path/has/got
96 1550611510 1550611458 1550611458 41ed /var/log/path/has/got/some
`
	depth := 3
	exec := suite.createExec(firstFixture, depth)
	cmd := StatCmd("/var/log/path", depth)
	exec.On("Exec", mock.Anything, cmd[0], cmd[1:], plugin.ExecOptions{Elevate: true}).Return(suite.createResult(secondFixture), nil)

	fs := NewFS("fs", exec, depth)
	// ID would normally be set when listing FS within the parent instance.
	fs.SetTestID("/instance/fs")

	entry := suite.find(fs, "var/log").(plugin.Parent)
	entries, err := plugin.List(suite.ctx, entry)
	if !suite.NoError(err) {
		suite.FailNow("Listing entries failed")
	}
	suite.Equal(1, entries.Len())
	suite.Contains(entries.Map(), "path")

	entries1, err := plugin.List(suite.ctx, entries.Map()["path"].(plugin.Parent))
	if suite.NoError(err) {
		suite.Equal(1, entries1.Len())
		suite.Contains(entries1.Map(), "has")
		entries2, err := plugin.List(suite.ctx, entries1.Map()["has"].(plugin.Parent))
		if suite.NoError(err) {
			suite.Equal(1, entries2.Len())
			suite.Contains(entries2.Map(), "got")
		}
	}
}

func (suite *fsTestSuite) TestFSListExpiredCache() {
	shortFixture := `
96 1550611510 1550611448 1550611448 41ed /var
96 1550611510 1550611448 1550611448 41ed /var/log
96 1550611510 1550611448 1550611448 41ed /var/log/path
`
	depth := 3
	exec := &mockExecutor{EntryBase: plugin.NewEntry("instance")}
	exec.SetTestID("/instance")
	cmd := StatCmd("/", depth)
	exec.On("Exec", mock.Anything, cmd[0], cmd[1:], plugin.ExecOptions{Elevate: true}).Return(mockExecCmd{shortFixture}, nil)

	fs := NewFS("fs", exec, depth)
	// ID would normally be set when listing FS within the parent instance.
	fs.SetTestID("/instance/fs")

	entry := suite.find(fs, "var/log").(plugin.Parent)
	entries, err := plugin.List(suite.ctx, entry)
	if !suite.NoError(err) {
		suite.FailNow("Listing entries failed")
	}
	suite.Equal(1, entries.Len())
	suite.Contains(entries.Map(), "path")

	_ = plugin.ClearCacheFor("/instance/fs")
	entries, err = plugin.List(suite.ctx, entry)
	if suite.NoError(err) {
		suite.Equal(1, entries.Len())
	}
	suite.Implements((*plugin.Parent)(nil), suite.find(fs, "var/log"))
}

func (suite *fsTestSuite) TestFSRead() {
	exec := suite.createExec(varLogFixture, varLogDepth)
	fs := NewFS("fs", exec, varLogDepth)
	// ID would normally be set when listing FS within the parent instance.
	fs.SetTestID("/instance/fs")

	entry := suite.find(fs, "var/log/path1/a file")
	suite.Equal("a file", plugin.Name(entry))

	execResult := suite.createResult("hello")
	exec.On("Exec", mock.Anything, "cat", []string{"/var/log/path1/a file"}, plugin.ExecOptions{Elevate: true}).Return(execResult, nil)
	rdr, err := entry.(plugin.Readable).Open(context.Background())
	suite.NoError(err)
	suite.Equal(int64(5), rdr.Size())
	buf := make([]byte, 5)
	n, err := rdr.ReadAt(buf, 0)
	suite.NoError(err)
	suite.Equal(5, n)
	suite.Equal("hello", string(buf))
}

func (suite *fsTestSuite) TestVolumeDelete() {
	exec := suite.createExec(varLogFixture, varLogDepth)
	fs := NewFS("fs", exec, varLogDepth)
	// ID would normally be set when listing FS within the parent instance.
	fs.SetTestID("/instance/fs")

	execResult := suite.createResult("deleted")
	exec.On("Exec", mock.Anything, "rm", []string{"-rf", "/var/log/path1/a file"}, plugin.ExecOptions{Elevate: true}).Return(execResult, nil)
	deleted, err := fs.VolumeDelete(context.Background(), "/var/log/path1/a file")
	if suite.NoError(err) {
		suite.True(deleted)
		exec.AssertCalled(suite.T(), "Exec", mock.Anything, "rm", []string{"-rf", "/var/log/path1/a file"}, plugin.ExecOptions{Elevate: true})
	}
}

func TestFS(t *testing.T) {
	suite.Run(t, new(fsTestSuite))
}

type mockExecutor struct {
	plugin.EntryBase
	mock.Mock
}

func (m *mockExecutor) Exec(ctx context.Context, cmd string, args []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	arger := m.Called(ctx, cmd, args, opts)
	return arger.Get(0).(plugin.ExecCommand), arger.Error(1)
}

func (m *mockExecutor) Schema() *plugin.EntrySchema {
	return nil
}

// Mock ExecCommand that can be used repeatedly when mocking a repeated call.
type mockExecCmd struct {
	data string
}

func (cmd mockExecCmd) OutputCh() <-chan plugin.ExecOutputChunk {
	ch := make(chan plugin.ExecOutputChunk, 1)
	ch <- plugin.ExecOutputChunk{
		StreamID:  plugin.Stdout,
		Timestamp: time.Now(),
		Data:      cmd.data,
	}
	close(ch)
	return ch
}

func (cmd mockExecCmd) ExitCode() (int, error) {
	return 0, nil
}
