package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

func TestAtime(t *testing.T) {
	s := newTimeAttrTestSuite("atime", Atime, func(e *rql.Entry, t time.Time) {
		e.Attributes.SetAtime(t)
	})
	suite.Run(t, s)
}
