package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

func TestMtime(t *testing.T) {
	s := newTimeAttrTestSuite("mtime", Mtime, func(e *rql.Entry, t time.Time) {
		e.Attributes.SetMtime(t)
	})
	suite.Run(t, s)
}
