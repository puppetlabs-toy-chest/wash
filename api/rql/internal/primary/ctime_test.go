package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

type CtimeTestSuite struct {
	TimeAttrTestSuite
}

func TestCtime(t *testing.T) {
	s := new(CtimeTestSuite)
	s.name = "ctime"
	s.constructP = Ctime
	s.setAttr = func(e *rql.Entry, t time.Time) {
		e.Attributes.SetCtime(t)
	}
	suite.Run(t, s)
}
