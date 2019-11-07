package cmdtest

import (
	"io"

	"github.com/stretchr/testify/mock"

	"github.com/puppetlabs/wash/analytics"
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

// Schema mocks Client#Schema
func (c *MockClient) Schema(path string) (*apitypes.EntrySchema, error) {
	args := c.Called(path)
	return args.Get(0).(*apitypes.EntrySchema), args.Error(1)
}

// Screenview mocks Client#Screenview
func (c *MockClient) Screenview(name string, params analytics.Params) error {
	args := c.Called(name, params)
	return args.Error(1)
}

// Delete mocks Client#Delete
func (c *MockClient) Delete(path string) (bool, error) {
	args := c.Called(path)
	return args.Get(0).(bool), args.Error(1)
}

// Signal mocks Client#Signal
func (c *MockClient) Signal(path string, signal string) error {
	args := c.Called(path, signal)
	return args.Error(0)
}
