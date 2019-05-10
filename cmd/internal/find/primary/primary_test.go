package primary

import (
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
)

// This file contains common test code for the primaries

type primaryTestSuite struct {
	parsertest.Suite
	ConstructEntry func(v interface{}) types.Entry
}

func (s *primaryTestSuite) NPETC(input string, errRegex string) parsertest.Case {
	return s.Suite.NPETC(input, errRegex, false)
}

func (s *primaryTestSuite) NPTC(input string, remInput string, trueValue interface{}) parsertest.Case {
	return s.Suite.NPTC(input, remInput, s.ConstructEntry(trueValue))
}

func (s *primaryTestSuite) NPNTC(input string, remInput string, falseValue interface{}) parsertest.Case {
	return s.Suite.NPNTC(input, remInput, s.ConstructEntry(falseValue))
}

// RTC => RunTestCase.
//
// TODO: May be worth changing NPTC/NPNTC/NPETC => RPTC/RPNTC/RPETC in the toplevel parser test suite.
func (s *primaryTestSuite) RTC(input string, remInput string, trueValue interface{}, falseValue interface{}) {
	s.RunTestCases(
		s.NPTC(input, remInput, trueValue),
		s.NPNTC(input, remInput, falseValue),
	)
}
