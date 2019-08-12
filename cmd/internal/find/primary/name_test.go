package primary

import (
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type NamePrimaryTestSuite struct {
	primaryTestSuite
}

func (s *NamePrimaryTestSuite) TestErrors() {
	s.RETC("", "requires additional arguments")
	s.RETC("[a", "invalid pattern: unexpected end of input")
}

func (s *NamePrimaryTestSuite) TestValidInput() {
	s.RTC("a", "", "a", "b")
}

func TestNamePrimary(t *testing.T) {
	s := new(NamePrimaryTestSuite)
	s.Parser = Name
	s.ConstructEntry = func(v interface{}) types.Entry {
		e := types.Entry{}
		e.CName = v.(string)
		return e
	}
	suite.Run(t, s)
}
