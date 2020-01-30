package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

type MtimeTestSuite struct {
	TimeAttrTestSuite
}

func TestMtime(t *testing.T) {
	s := new(MtimeTestSuite)
	s.name = "mtime"
	s.constructP = Mtime
	s.setAttr = func(e *rql.Entry, t time.Time) {
		e.Attributes.SetMtime(t)
	}
	suite.Run(t, s)
}
