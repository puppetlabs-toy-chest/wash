package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HelpersTestSuite struct {
	suite.Suite
}

func (suite *HelpersTestSuite) TestToMetadata() {
	cases := []struct {
		input    interface{}
		expected MetadataMap
	}{
		{[]byte(`{"hello": [1, 2, 3]}`), MetadataMap{"hello": []interface{}{1.0, 2.0, 3.0}}},
		{struct {
			Name  string
			Value []int
		}{"me", []int{1, 2, 3}}, MetadataMap{"Name": "me", "Value": []interface{}{1.0, 2.0, 3.0}}},
	}
	for _, c := range cases {
		actual := ToMetadata(c.input)
		suite.Equal(c.expected, actual)
	}
}

func (suite *HelpersTestSuite) TestParseMode() {
	type testCase struct {
		input    interface{}
		expected uint64
		errRegex string
	}

	cases := []testCase{
		{input: uint64(10), expected: 10},
		{input: int64(10), expected: 10},
		{input: float64(10.0), expected: 10},
		{input: float64(10.5), errRegex: "decimal.*number"},
		{input: []byte("invalid mode type"), errRegex: "uint64.*int64.*float64.*string"},
		{input: "15", expected: 15},
		{input: "0777", expected: 511},
		{input: "0xf", expected: 15},
		{input: "not a number", errRegex: "not a number"},
	}

	for _, c := range cases {
		actual, err := parseMode(c.input)
		if c.errRegex != "" {
			suite.Regexp(regexp.MustCompile(c.errRegex), err)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, actual)
			}
		}
	}
}

func (suite *HelpersTestSuite) TestToFileMode() {
	type testCase struct {
		input    interface{}
		expected os.FileMode
		errRegex string
	}

	cases := []testCase{
		{input: "not a number", errRegex: "not a number"},
		// 16877 is 0x41ed in decimal
		{input: "0x41ed", expected: 0755 | os.ModeDir},
		{input: float64(16877), expected: 0755 | os.ModeDir},
		// 33188 is 0x81a4 in decimal
		{input: "0x81a4", expected: 0644},
		{input: float64(33188), expected: 0644},
	}

	for _, c := range cases {
		actual, err := ToFileMode(c.input)
		if c.errRegex != "" {
			suite.Regexp(regexp.MustCompile(c.errRegex), err)
		} else {
			if suite.NoError(err) {
				suite.Equal(c.expected, actual)
			}
		}
	}
}

type mockEntry struct {
	EntryBase
	mock.Mock
}

func (e *mockEntry) Attr() Attributes {
	args := e.Called()
	return args.Get(0).(Attributes)
}

type mockGroup struct {
	EntryBase
	mock.Mock
}

func (e *mockGroup) List(ctx context.Context) ([]Entry, error) {
	args := e.Called()
	return args.Get(0).([]Entry), args.Error(1)
}

func (suite *HelpersTestSuite) TestFillAttrAnyEntry() {
	// Test that Attr()'s result is used if the entry implements it
	// and if entry.Attr() returns a non-zero set of attributes
	entry := &mockEntry{EntryBase: NewEntry("mockFileEntry")}
	expectedAttributes := Attributes{
		Ctime: time.Now(),
		Size:  10,
		Mode:  0777,
	}
	entry.On("Attr").Return(expectedAttributes)
	attr := Attributes{}
	err := FillAttr(context.Background(), entry, "id", &attr)
	if suite.NoError(err) {
		suite.Equal(
			expectedAttributes,
			attr,
			"FillAttr should use the entry's filesystem attributes if they're provided",
		)
	}

	// Test that if the entry supports the list action, then it sets
	// the mode to 0550
	group := &mockGroup{EntryBase: NewEntry("mockGroup")}
	attr = Attributes{}
	err = FillAttr(context.Background(), group, "id", &attr)
	if suite.NoError(err) {
		suite.Equal(
			Attributes{Mode: os.ModeDir | 0550, Size: SizeUnknown},
			attr,
			"FillAttr does not set the default mode to os.ModeDir | 0550 for entries that support the list action",
		)
	}

	// Test that if the entry does not support the list action, then it sets
	// the mode to 0440
	entry = &mockEntry{EntryBase: NewEntry("mockEntry")}
	entry.On("Attr").Return(Attributes{})
	attr = Attributes{}
	err = FillAttr(context.Background(), entry, "id", &attr)
	if suite.NoError(err) {
		suite.Equal(
			Attributes{Mode: 0440, Size: SizeUnknown},
			attr,
			"FillAttr does not set the default mode to 0440 for entries that do notsupport the list action",
		)
	}
}

type mockReadableEntry struct {
	*mockEntry
}

func (e *mockReadableEntry) Open(ctx context.Context) (SizedReader, error) {
	args := e.Called(ctx)
	return args.Get(0).(SizedReader), args.Error(1)
}

func newMockReadableEntry(attr Attributes) *mockReadableEntry {
	entry := new(mockReadableEntry)
	entry.mockEntry = &mockEntry{EntryBase: NewEntry("mockReadableEntry")}
	entry.On("Attr").Return(attr)
	return entry
}

type negativeSizedReader struct{}

func (r negativeSizedReader) ReadAt(b []byte, off int64) (int, error) {
	return 0, fmt.Errorf("negativeSizedReader is not meant to read anything")
}

func (r negativeSizedReader) Size() int64 {
	return -1
}

func (suite *HelpersTestSuite) TestFillAttrReadableEntry() {
	ctx := context.Background()
	mockRdr := strings.NewReader("foo")

	// Test that the size attribute is calculated from the content
	// if it is not known
	entry := newMockReadableEntry(Attributes{})
	entry.On("Open", ctx).Return(mockRdr, nil)
	attr := Attributes{}
	err := FillAttr(ctx, entry, "id", &attr)
	if suite.NoError(err) {
		suite.Equal(
			uint64(mockRdr.Size()),
			attr.Size,
			"FillAttr should calculate the size attribute from the entry's content if its value is SizeUnknown",
		)
	}

	// Test that the size attribute is _not_ calculated from the content
	// if it is already known
	entry = newMockReadableEntry(Attributes{Size: 10})
	entry.On("Open", ctx).Return(mockRdr, nil)
	attr = Attributes{}
	err = FillAttr(ctx, entry, "id", &attr)
	if suite.NoError(err) {
		suite.Equal(
			uint64(10),
			attr.Size,
			"FillAttr should _not_ calculate the size attribute from the entry's content if it is already known",
		)
	}

	// Test that FillAttr returns an ErrCouldNotDetermineSizeAttr error if
	// Open errors, and that it still proceeds to fill-in the mode for FUSE
	entry = newMockReadableEntry(Attributes{})
	entry.On("Open", ctx).Return(mockRdr, fmt.Errorf("could not open"))
	attr = Attributes{}
	err = FillAttr(ctx, entry, "id", &attr)
	suite.EqualError(err, ErrCouldNotDetermineSizeAttr{"could not open"}.Error())
	suite.Equal(
		os.FileMode(0440),
		attr.Mode,
		"FillAttr should fill-in the Mode attribute for FUSE even when Open errors",
	)

	// Test that FillAttr returns an ErrNegativeSizeAttr error if the content
	// has negative size
	entry = newMockReadableEntry(Attributes{})
	entry.On("Open", ctx).Return(negativeSizedReader{}, nil)
	attr = Attributes{}
	err = FillAttr(ctx, entry, "id", &attr)
	suite.EqualError(err, ErrNegativeSizeAttr{-1}.Error())
}

func (suite *HelpersTestSuite) TestExitCodeFromErr() {
	exitCode, err := ExitCodeFromErr(nil)
	if suite.NoError(err) {
		suite.Equal(
			0,
			exitCode,
			"ExitCodeFromErr should return an exit code of 0 if no error was passed-in",
		)
	}

	arbitraryErr := fmt.Errorf("an arbitrary error")
	_, err = ExitCodeFromErr(arbitraryErr)
	suite.EqualError(err, arbitraryErr.Error())

	// The default exit code is 0 for an empty ProcessState object
	exitErr := &exec.ExitError{ProcessState: &os.ProcessState{}}
	exitCode, err = ExitCodeFromErr(exitErr)
	if suite.NoError(err) {
		suite.Equal(
			0,
			exitCode,
			"ExitCodeFromErr should return the ExitError's exit code",
		)
	}
}

func (suite *HelpersTestSuite) SetupTest() {
	// Turn off the cache in case another set of tests initialized it
	cache = nil
}

func TestHelpers(t *testing.T) {
	suite.Run(t, new(HelpersTestSuite))
}
