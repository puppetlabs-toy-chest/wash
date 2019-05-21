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

func (s *primaryTestSuite) RETC(input string, errRegex string) {
	s.Suite.RETC(input, errRegex, false)
}

func (s *primaryTestSuite) RTC(input string, remInput string, trueValue interface{}, falseValue ...interface{}) {
	s.Suite.RTC(input, remInput, s.ConstructEntry(trueValue))
	if len(falseValue) > 0 {
		s.Suite.RNTC(input, remInput, s.ConstructEntry(falseValue[0]))
	}
}

func (s *primaryTestSuite) RNTC(input string, remInput string, falseValue interface{}) {
	s.Suite.RNTC(input, remInput, s.ConstructEntry(falseValue))
}
