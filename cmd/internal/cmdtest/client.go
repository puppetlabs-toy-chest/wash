package cmdtest

import (
	"io"

	"github.com/stretchr/testify/mock"

	apitypes "github.com/puppetlabs/wash/api/types"
)

// MockClient mocks a Wash API client
type MockClient struct {
	mock.Mock
}

// Info mocks Client#Info
func (c *MockClient) Info(path string) (apitypes.Entry, error) {
	args := c.Called(path)
	return args.Get(0).(apitypes.Entry), args.Error(1)
}

// List mocks Client#List
func (c *MockClient) List(path string) ([]apitypes.Entry, error) {
	args := c.Called(path)
	return args.Get(0).([]apitypes.Entry), args.Error(1)
}

// Metadata mocks Client#Metadata
func (c *MockClient) Metadata(path string) (map[string]interface{}, error) {
	args := c.Called(path)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

// Stream mocks Client#Stream
func (c *MockClient) Stream(path string) (io.ReadCloser, error) {
	args := c.Called(path)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Exec mocks Client#Exec
func (c *MockClient) Exec(path string, command string, args []string, opts apitypes.ExecOptions) (<-chan apitypes.ExecPacket, error) {
	margs := c.Called(path, command, args, opts)
	return margs.Get(0).(<-chan apitypes.ExecPacket), margs.Error(1)
}

// History mocks Client#History
func (c *MockClient) History(follow bool) (chan apitypes.Activity, error) {
	args := c.Called(follow)
	return args.Get(0).(chan apitypes.Activity), args.Error(1)
}

// ActivityJournal mocks Client#ActivityJournal
func (c *MockClient) ActivityJournal(index int, follow bool) (io.ReadCloser, error) {
	args := c.Called(index, follow)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

// Clear mocks Client#Clear
func (c *MockClient) Clear(path string) ([]string, error) {
	args := c.Called(path)
	return args.Get(0).([]string), args.Error(1)
}