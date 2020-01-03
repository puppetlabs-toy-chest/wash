package plugintest

import (
	"context"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
)

// MockBase is a basic mock with no read/write operations.
type MockBase struct {
	plugin.EntryBase
	mock.Mock
}

// NewMockBase creates a new "mock" entry without read/write.
func NewMockBase() *MockBase {
	m := &MockBase{EntryBase: plugin.NewEntry("mock")}
	m.SetTestID("/mock")
	return m
}

// Schema returns a simple schema.
func (m *MockBase) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(m, "mock")
}

var _ = plugin.Entry(&MockBase{})

// MockRead only mocks Read operations.
type MockRead struct {
	MockBase
}

// NewMockRead creates a new "mock" entry for reads.
func NewMockRead() *MockRead {
	m := &MockRead{MockBase{EntryBase: plugin.NewEntry("mockr")}}
	m.SetTestID("/mockr")
	return m
}

func (m *MockRead) Read(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([]byte), args.Error(1)
}

var _ = plugin.Readable(&MockRead{})

// MockWrite only mocks Write operations.
type MockWrite struct {
	MockBase
}

// NewMockWrite creates a new "mock" entry for writes.
func NewMockWrite() *MockWrite {
	m := &MockWrite{MockBase{EntryBase: plugin.NewEntry("mockw")}}
	m.SetTestID("/mockw")
	return m
}

func (m *MockWrite) Write(ctx context.Context, p []byte) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

var _ = plugin.Writable(&MockWrite{})

// MockReadWrite mocks read and write operations.
type MockReadWrite struct {
	MockBase
}

// NewMockReadWrite creates a new "mock" entry for reads and writes.
func NewMockReadWrite() *MockReadWrite {
	m := &MockReadWrite{MockBase{EntryBase: plugin.NewEntry("mockrw")}}
	m.SetTestID("/mockrw")
	return m
}

func (m *MockReadWrite) Read(ctx context.Context) ([]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockReadWrite) Write(ctx context.Context, p []byte) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

var _ = plugin.Readable(&MockReadWrite{})
var _ = plugin.Writable(&MockReadWrite{})

// MockBlockReadWrite mocks block read and write operations.
type MockBlockReadWrite struct {
	MockBase
}

// NewMockBlockReadWrite creates a new "mock" entry for (block) reads and writes.
func NewMockBlockReadWrite() *MockBlockReadWrite {
	m := &MockBlockReadWrite{MockBase{EntryBase: plugin.NewEntry("mockbrw")}}
	m.SetTestID("/mockbrw")
	return m
}

func (m *MockBlockReadWrite) Read(ctx context.Context, size, off int64) ([]byte, error) {
	args := m.Called(ctx, size, off)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockBlockReadWrite) Write(ctx context.Context, p []byte) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

var _ = plugin.BlockReadable(&MockBlockReadWrite{})
var _ = plugin.Writable(&MockBlockReadWrite{})
