package primary

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type SizePrimaryTestSuite struct {
	suite.Suite
}

func (suite *SizePrimaryTestSuite) TestSizePrimaryInsufficientArgsError() {
	_, _, err := sizePrimary.Parse([]string{"-size"})
	suite.Equal("-size: requires additional arguments", err.Error())
}

func (suite *SizePrimaryTestSuite) TestSizePrimaryIllegalTimeValueError() {
	illegalValues := []string{
		"foo",
		"+",
		"+++++1",
		"1kb",
		"+1kb",
	}
	for _, v := range illegalValues {
		_, _, err := sizePrimary.Parse([]string{"-size", v})
		msg := fmt.Sprintf("-size: %v: illegal size value", v)
		suite.Equal(msg, err.Error())
	}
}

func (suite *SizePrimaryTestSuite) TestSizePrimaryValidInput() {
	type testCase struct {
		input string
		// trueSize/falseSize represent entry sizes that satisfy/unsatisfy
		// the predicate, respectively.
		trueSize  int64
		falseSize int64
	}
	testCases := []testCase{
		// We set trueSize to 1.5 blocks in order to test rounding
		testCase{"2", int64(1.5 * 512), 512},
		// +2 means p will return true if size > 2 blocks
		testCase{"+2", 3 * 512, 1 * 512},
		// -2 means p will return true if size < 2 blocks
		testCase{"-2", 1 * 512, 2 * 512},
		testCase{"1k", 1 * numeric.BytesOf('k'), 1 * numeric.BytesOf('c')},
		testCase{"+1k", 2 * numeric.BytesOf('k'), 1 * numeric.BytesOf('k')},
		testCase{"-1k", 1 * numeric.BytesOf('c'), 1 * numeric.BytesOf('k')},
	}
	for _, testCase := range testCases {
		inputStr := func() string {
			return fmt.Sprintf("Input was '%v'", testCase.input)
		}
		p, tokens, err := sizePrimary.Parse([]string{"-size", testCase.input})
		if suite.NoError(err, inputStr()) {
			suite.Equal([]string{}, tokens)
			e := types.Entry{}
			// Ensure p(e) is always false for an entry that doesn't have a size attribute
			suite.False(p(e), inputStr())

			e.Attributes.SetSize(uint64(testCase.trueSize))
			suite.True(p(e), inputStr())

			e.Attributes.SetSize(uint64(testCase.falseSize))
			suite.False(p(e), inputStr())
		}
	}
}

func TestSizePrimary(t *testing.T) {
	suite.Run(t, new(SizePrimaryTestSuite))
}
