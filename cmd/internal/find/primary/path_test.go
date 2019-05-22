package primary

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type PathPrimaryTestSuite struct {
	primaryTestSuite
}

func (s *PathPrimaryTestSuite) TestErrors() {
	s.RETC("", "requires additional arguments")
	s.RETC("[a", "invalid glob: unexpected end of input")
}

func (s *PathPrimaryTestSuite) TestValidInput() {
	s.RTC("a", "", "a", "b")
}

func TestPathPrimary(t *testing.T) {
	s := new(PathPrimaryTestSuite)
	s.Parser = Path
	s.ConstructEntry = func(v interface{}) types.Entry {
		e := types.Entry{}
		e.NormalizedPath = v.(string)
		return e
	}
	suite.Run(t, s)
}
