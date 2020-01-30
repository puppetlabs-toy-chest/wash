package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

type AtimeTestSuite struct {
	TimeAttrTestSuite
}

func TestAtime(t *testing.T) {
	s := new(AtimeTestSuite)
	s.name = "atime"
	s.constructP = Atime
	s.setAttr = func(e *rql.Entry, t time.Time) {
		e.Attributes.SetAtime(t)
	}
	suite.Run(t, s)
}
