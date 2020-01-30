package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

type CrtimeTestSuite struct {
	TimeAttrTestSuite
}

func TestCrtime(t *testing.T) {
	s := new(CrtimeTestSuite)
	s.name = "crtime"
	s.constructP = Crtime
	s.setAttr = func(e *rql.Entry, t time.Time) {
		e.Attributes.SetCrtime(t)
	}
	suite.Run(t, s)
}
