package primary

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/primary/numeric"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type SizePrimaryTestSuite struct {
	primaryTestSuite
}

func (s *SizePrimaryTestSuite) TestErrors() {
	// NIVTC => NewIllegalValueTestCase
	NIVTC := func(v string) parsertest.Case {
		return s.NPETC(v, fmt.Sprintf("%v: illegal size value", regexp.QuoteMeta(v)))
	}
	s.RunTestCases(
		s.NPETC("", "requires additional arguments"),
		NIVTC("foo"),
		NIVTC("+"),
		NIVTC("+++++1"),
		NIVTC("+1kb"),
		NIVTC("+1kb"),
	)
}

func (s *SizePrimaryTestSuite) TestValidInput() {
	// We set the size to 1.5 blocks in order to test rounding
	s.RTC("2", "", int64(1.5 * 512), int64(512))
	// +2 means p will return true if size > 2 blocks
	s.RTC("+2", "", int64(3 * 512), int64(1 * 512))
	// -2 means p will return true if size < 2 blocks
	s.RTC("-2", "", int64(1 * 512), int64(2 * 512))
	s.RTC("1k", "", 1 * numeric.BytesOf('k'), 1 * numeric.BytesOf('c'))
	s.RTC("+1k", "", 2 * numeric.BytesOf('k'), 1 * numeric.BytesOf('k'))
	s.RTC("-1k", "", 1 * numeric.BytesOf('c'), 1 * numeric.BytesOf('k'))
}

func TestSizePrimary(t *testing.T) {
	s := new(SizePrimaryTestSuite)
	s.Parser = sizePrimary
	s.ConstructEntry = func(v interface{}) types.Entry {
		e := types.Entry{}
		e.Attributes.SetSize(uint64(v.(int64)))
		return e
	}
	suite.Run(t, s)
}
