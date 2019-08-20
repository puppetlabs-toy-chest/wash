package primary

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type ParserTestSuite struct {
	parsertest.Suite
}

func (s *ParserTestSuite) TestRegisteredPrimaries() {
	expectedList := []*Primary{
		Action,
		True,
		False,
		Meta,
		Name,
		Path,
		Size,
		Ctime,
		Mtime,
		Atime,
		Crtime,
		Kind,
	}
	expectedMp := map[string]*Primary{
		"-action": Action,
		"-true":   True,
		"-false":  False,
		"-meta":   Meta,
		"-m":      Meta,
		"-name":   Name,
		"-path":   Path,
		"-size":   Size,
		"-ctime":  Ctime,
		"-mtime":  Mtime,
		"-atime":  Atime,
		"-crtime": Crtime,
		"-kind":   Kind,
		"-k":      Kind,
	}

	s.ElementsMatch(expectedList, Parser.primaries)
	s.Equal(expectedMp, Parser.primaryMap)
}

func (s *ParserTestSuite) TestErrors() {
	s.RETC("", "expected a primary", true)
	s.RETC("-foo", "foo: unknown primary", true)
	// Test that a primary parse error is printed as "<primary token>: <err>"
	s.RETC("-meta", "-meta: expected a key sequence", false)
	s.RETC("-m", "-m: expected a key sequence", false)
}

func (s *ParserTestSuite) TestValidInput() {
	e := types.Entry{}
	e.CName = "a"
	s.RTC("-name a", "", e)
}

func TestPrimaryParser(t *testing.T) {
	// Use the -name primary as a representative case
	s := new(ParserTestSuite)
	s.Parser = Parser
	suite.Run(t, s)
}
