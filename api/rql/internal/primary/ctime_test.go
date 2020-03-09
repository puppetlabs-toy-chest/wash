package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

func TestCtime(t *testing.T) {
	s := newTimeAttrTestSuite("ctime", Ctime, func(e *rql.Entry, t time.Time) {
		e.Attributes.SetCtime(t)
	})
	suite.Run(t, s)
}
