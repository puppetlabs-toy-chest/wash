package plugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EntryContentTestSuite struct {
	suite.Suite
}

func (s *EntryContentTestSuite) TestEntryContentImpl() {
	// EntryContentImpl will likely never change, so go ahead and group
	// all its tests here
	rawContent := []byte("some raw content")
	contentSize := int64(len(rawContent))
	content := newEntryContent(rawContent)

	// Test entryContentImpl#size
	s.Equal(uint64(len(rawContent)), content.size())

	// Now test entryContentImpl#read
	type testCase struct {
		size     int64
		offset   int64
		expected []byte
	}
	testCases := []testCase{
		// Test offset >= contentSize
		testCase{size: 0, offset: contentSize, expected: []byte("")},
		testCase{size: 0, offset: contentSize + 1, expected: []byte("")},
		// Test happy-cases
		testCase{size: 0, offset: 0, expected: []byte("")},
		testCase{size: 0, offset: 1, expected: []byte("")},
		testCase{size: 4, offset: 2, expected: []byte("me r")},
		testCase{size: contentSize, offset: 0, expected: rawContent},
		// Test out-of-bounds sizes
		testCase{size: contentSize + 1, offset: 0, expected: rawContent},
		testCase{size: contentSize, offset: 1, expected: rawContent[1:]},
	}
	for _, testCase := range testCases {
		actual, err := content.read(context.Background(), testCase.size, testCase.offset)
		if s.NoError(err) {
			s.Equal(testCase.expected, actual)
		}
	}
}

func (s *EntryContentTestSuite) TestBlockReadableEntryContent_Size() {
	content := newBlockReadableEntryContent(func(_ context.Context, _ int64, _ int64) ([]byte, error) {
		return nil, nil
	})
	content.sz = 10
	s.Equal(uint64(10), content.size())
}

type mockBlockReadableEntry struct {
	mock.Mock
}

func (m *mockBlockReadableEntry) Read(ctx context.Context, size int64, offset int64) ([]byte, error) {
	args := m.Called(ctx, size, offset)
	return args.Get(0).([]byte), args.Error(1)
}

func (s *EntryContentTestSuite) TestBlockReadableEntryContent_Read_SuccessfulReadFuncInvocation() {
	m := &mockBlockReadableEntry{}
	content := newBlockReadableEntryContent(func(ctx context.Context, size int64, offset int64) ([]byte, error) {
		return m.Read(ctx, size, offset)
	})

	ctx := context.Background()
	m.On("Read", ctx, int64(10), int64(0)).Return([]byte("some raw content"), nil).Once()
	rawContent, err := content.read(ctx, 10, 0)
	if s.NoError(err) {
		s.Equal([]byte("some raw content"), rawContent)
	}
}

func (s *EntryContentTestSuite) TestBlockReadableEntryContent_Read_ErroredReadFuncInvocation() {
	m := &mockBlockReadableEntry{}
	content := newBlockReadableEntryContent(func(ctx context.Context, size int64, offset int64) ([]byte, error) {
		return m.Read(ctx, size, offset)
	})

	ctx := context.Background()
	expectedErr := fmt.Errorf("an error")
	m.On("Read", ctx, int64(10), int64(0)).Return([]byte{}, expectedErr).Once()

	_, err := content.read(ctx, 10, 0)
	s.Equal(expectedErr, err)
}

func TestEntryContent(t *testing.T) {
	suite.Run(t, new(EntryContentTestSuite))
}
