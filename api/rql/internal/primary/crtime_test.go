package primary

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/stretchr/testify/suite"
)

func TestCrtime(t *testing.T) {
	s := newTimeAttrTestSuite("crtime", Crtime, func(e *rql.Entry, t time.Time) {
		e.Attributes.SetCrtime(t)
	})
	suite.Run(t, s)
}
