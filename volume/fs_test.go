package volume

import (
	"bytes"
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

const fixtureDepth = 7

// Represents the output of StatCmdPOSIX(/var/log)
const (
	posixFixture = `
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
	posixFixtureShort = `
96 1550611510 1550611448 1550611448 41ed /var
96 1550611510 1550611448 1550611448 41ed /var/log
96 1550611510 1550611448 1550611448 41ed /var/log/path
`
	posixFixtureDeep = `
96 1550611510 1550611448 1550611448 41ed /var/log/path/has
96 1550611510 1550611448 1550611448 41ed /var/log/path/has/got
96 1550611510 1550611458 1550611458 41ed /var/log/path/has/got/some
`
)

// Represents the output of StatCmdPowershell(/var/log)
const (
	powershellFixture = `
"FullName","Length","CreationTimeUtc","LastAccessTimeUtc","LastWriteTimeUtc","Attributes"
"C:\var\log\path",,"2018-09-15T07:19:00Z","2020-01-07T21:11:01Z","2020-01-07T21:10:43Z","Directory"
"C:\var\log\path\has",,"2018-09-15T06:09:26Z","2020-01-07T20:43:01Z","2020-01-07T20:43:01Z","Directory"
"C:\var\log\path\has\got",,"2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","Directory"
"C:\var\log\path\has\got\some",,"2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","Directory"
"C:\var\log\path\has\got\some\legs","0","2018-09-15T07:19:01Z","2019-09-07T00:21:10Z","2019-09-07T00:21:10Z","ReadOnly, Archive"
"C:\var\log\path1",,"2018-09-15T07:19:00Z","2018-09-15T07:19:03Z","2018-09-15T07:19:03Z","Directory"
"C:\var\log\path1\a file","7842","2019-10-13T08:15:00Z","2019-10-13T08:15:00Z","2019-10-13T08:15:00Z","NotContentIndexed, Archive"
"C:\var\log\path2",,"2018-09-15T07:12:58Z","2018-09-15T07:12:58Z","2018-09-15T07:12:58Z","System, Directory"
"C:\var\log\path2\dir","67584","2019-10-13T01:16:07Z","2020-01-07T21:05:03Z","2020-01-07T21:05:03Z","Directory, System"
`
	powershellFixtureShort = `
"C:\var",,"2018-09-15T07:19:00Z","2020-01-07T21:11:01Z","2020-01-07T21:10:43Z","Directory"
"C:\var\log",,"2018-09-15T07:19:00Z","2020-01-07T21:11:01Z","2020-01-07T21:10:43Z","Directory"
"C:\var\log\path",,"2018-09-15T07:19:00Z","2020-01-07T21:11:01Z","2020-01-07T21:10:43Z","Directory"
`
	powershellFixtureDeep = `
"C:\var\log\path\has",,"2018-09-15T06:09:26Z","2020-01-07T20:43:01Z","2020-01-07T20:43:01Z","Directory"
"C:\var\log\path\has\got",,"2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","Directory"
"C:\var\log\path\has\got\some",,"2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","2018-09-15T07:19:01Z","Directory"
`
)

type fsTestSuite struct {
	suite.Suite
	ctx                                context.Context
	cancelFunc                         context.CancelFunc
	loginShell                         plugin.Shell
	statCmd                            func(path string, maxdepth int) []string
	outputFixture                      string
	outputDepth                        int
	shortFixture, deepFixture          string
	readCmdFn, writeCmdFn, deleteCmdFn func(path string) (command []string)
}

func (suite *fsTestSuite) SetupTest() {
	// Use a different cache each test because we may re-use some but not all of the same structure.
	ctx := plugin.SetTestCache(datastore.NewMemCache())
	suite.ctx, suite.cancelFunc = context.WithCancel(ctx)
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

func (suite *fsTestSuite) createExec() *mockExecutor {
	exec := &mockExecutor{EntryBase: plugin.NewEntry("instance")}
	// Used when recording activity.
	exec.SetTestID("/instance")
	exec.Attributes().SetOS(plugin.OS{LoginShell: suite.loginShell})
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
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", suite.outputDepth), suite.createResult(suite.outputFixture))

	fs := NewFS(suite.ctx, "fs", exec, suite.outputDepth)

	entry := suite.find(fs, "var/log").(plugin.Parent)
	entries, err := entry.List(suite.ctx)
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

	entries1, err := entries[1].(plugin.Parent).List(suite.ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries1))
		suite.Equal("a file", plugin.Name(entries1[0]))
		_, ok := entries1[0].(plugin.Readable)
		suite.True(ok)
	}

	entries2, err := entries[2].(plugin.Parent).List(suite.ctx)
	if suite.NoError(err) {
		suite.Equal(1, len(entries2))
		suite.Equal("dir", plugin.Name(entries2[0]))
		_, ok := entries2[0].(plugin.Parent)
		suite.True(ok)
	}
	exec.AssertExpectations(suite.T())
}

func (suite *fsTestSuite) TestFSListTwice() {
	depth := 3
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", depth), suite.createResult(suite.shortFixture))
	exec.onExec(suite.statCmd("/var/log/path", depth), suite.createResult(suite.deepFixture))

	fs := NewFS(suite.ctx, "fs", exec, depth)

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
	exec.AssertExpectations(suite.T())
}

func (suite *fsTestSuite) TestFSListExpiredCache() {
	depth := 3
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", depth), mockExecCmd{suite.shortFixture}).Twice()

	fs := NewFS(suite.ctx, "fs", exec, depth)

	entry := suite.find(fs, "var/log").(plugin.Parent)
	entries, err := plugin.List(suite.ctx, entry)
	if !suite.NoError(err) {
		suite.FailNow("Listing entries failed")
	}
	suite.Equal(1, entries.Len())
	suite.Contains(entries.Map(), "path")

	cleared := plugin.ClearCacheFor("/fs", false)
	suite.Equal([]string{"List::/fs"}, cleared)
	entries, err = plugin.List(suite.ctx, entry)
	if suite.NoError(err) {
		suite.Equal(1, entries.Len())
	}
	suite.Implements((*plugin.Parent)(nil), suite.find(fs, "var/log"))
	exec.AssertExpectations(suite.T())
}

func (suite *fsTestSuite) TestFSRead() {
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", suite.outputDepth), suite.createResult(suite.outputFixture))

	fs := NewFS(suite.ctx, "fs", exec, suite.outputDepth)

	entry := suite.find(fs, "var/log/path1/a file")
	suite.Equal("a file", plugin.Name(entry))
	exec.onExec(suite.readCmdFn("/var/log/path1/a file"), suite.createResult("hello"))

	content, err := entry.(plugin.Readable).Read(suite.ctx)
	suite.NoError(err)
	suite.Equal([]byte("hello"), content)
	exec.AssertExpectations(suite.T())
}

func (suite *fsTestSuite) TestFSWrite() {
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", suite.outputDepth), suite.createResult(suite.outputFixture))

	fs := NewFS(suite.ctx, "fs", exec, suite.outputDepth)

	entry := suite.find(fs, "var/log/path1/a file")
	suite.Equal("a file", plugin.Name(entry))

	data := []byte("data")
	cmd := suite.writeCmdFn("/var/log/path1/a file")
	opts := plugin.ExecOptions{Elevate: true, Stdin: bytes.NewReader(data)}
	exec.On("Exec", mock.Anything, cmd[0], cmd[1:], opts).Return(suite.createResult(""), nil)

	err := entry.(plugin.Writable).Write(suite.ctx, data)
	suite.NoError(err)
	exec.AssertExpectations(suite.T())
}

func (suite *fsTestSuite) TestVolumeDelete() {
	exec := suite.createExec()
	exec.onExec(suite.statCmd("/", suite.outputDepth), suite.createResult(suite.outputFixture))
	fs := NewFS(suite.ctx, "fs", exec, suite.outputDepth)

	exec.onExec(suite.deleteCmdFn("/var/log/path1/a file"), suite.createResult("deleted"))
	deleted, err := fs.VolumeDelete(suite.ctx, "/var/log/path1/a file")
	suite.NoError(err)
	suite.True(deleted)
	exec.AssertExpectations(suite.T())
}

func TestPOSIXFS(t *testing.T) {
	suite.Run(t, &fsTestSuite{
		loginShell:    plugin.POSIXShell,
		statCmd:       StatCmdPOSIX,
		outputFixture: posixFixture,
		outputDepth:   fixtureDepth,
		shortFixture:  posixFixtureShort,
		deepFixture:   posixFixtureDeep,
		readCmdFn:     func(path string) []string { return []string{"cat", path} },
		writeCmdFn:    func(path string) []string { return []string{"cp", "/dev/stdin", path} },
		deleteCmdFn:   func(path string) []string { return []string{"rm", "-rf", path} },
	})
}

func TestPowershellFS(t *testing.T) {
	suite.Run(t, &fsTestSuite{
		loginShell:    plugin.PowerShell,
		statCmd:       StatCmdPowershell,
		outputFixture: powershellFixture,
		outputDepth:   fixtureDepth,
		shortFixture:  powershellFixtureShort,
		deepFixture:   powershellFixtureDeep,
		readCmdFn:     func(path string) []string { return []string{"Get-Content '" + path + "'"} },
		writeCmdFn:    func(path string) []string { return []string{"$input | Set-Content '" + path + "'"} },
		deleteCmdFn:   func(path string) []string { return []string{"Remove-Item -Recurse -Force '" + path + "'"} },
	})
}

type mockExecutor struct {
	plugin.EntryBase
	mock.Mock
}

func (m *mockExecutor) Exec(ctx context.Context, cmd string, args []string,
	opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	arger := m.Called(ctx, cmd, args, opts)
	return arger.Get(0).(plugin.ExecCommand), arger.Error(1)
}

func (m *mockExecutor) Schema() *plugin.EntrySchema {
	return nil
}

func (m *mockExecutor) onExec(cmd []string, result plugin.ExecCommand) *mock.Call {
	return m.On("Exec", mock.Anything, cmd[0], cmd[1:], mock.Anything).Return(result, nil)
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
