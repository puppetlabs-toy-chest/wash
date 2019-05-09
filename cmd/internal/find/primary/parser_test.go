package primary

import (
	"testing"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/stretchr/testify/suite"
)

type ParserTestSuite struct {
	parsertest.Suite
}

func (s *ParserTestSuite) TestErrors() {
	s.RunTestCases(
		s.NPETC("", "expected a primary", true),
		s.NPETC("-foo", "foo: unknown primary", true),
		// Test that a primary parse error is printed as "<primary token>: <err>"
		s.NPETC("-meta", "-meta: expected a key sequence", false),
		s.NPETC("-m", "-m: expected a key sequence", false),
	)
}

func (s *ParserTestSuite) TestValidInput() {
	e := types.Entry{}
	e.CName = "a"
	s.RunTestCases(
		s.NPTC("-name a", "", e),
	)
}

func TestPrimaryParser(t *testing.T) {
	// Use the -name primary as a representative case
	s := new(ParserTestSuite)
	s.Parser = Parser
	suite.Run(t, s)
}