package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/stretchr/testify/suite"
)

// These test Marshal/Unmarshal. Correctness of the Eval* methods
// is contained in the relevant predicate's unit tests to ensure
// that negation semantics are correct.

type NotTestSuite struct {
	asttest.Suite
}

func (s *NotTestSuite) TestMarshal() {
	s.MTC(Not(predicate.Boolean(true)), s.A("NOT", predicate.Boolean(true).Marshal()))
}

func (s *NotTestSuite) TestUnmarshal() {
	p := Not(predicate.Boolean(false))
	s.UMETC(p, "foo", "formatted.*'NOT'.*<pe>", true)
	s.UMETC(p, s.A("NOT", "foo", "bar"), "formatted.*'NOT'.*<pe>", false)
	s.UMETC(p, s.A("NOT"), "NOT.*expression", false)
	s.UMETC(p, s.A("NOT", s.A()), "NOT.*error.*expression.*formatted.*<boolean_value>", false)
	s.UMTC(p, s.A("NOT", true), Not(predicate.Boolean(true)))
}

func TestNot(t *testing.T) {
	suite.Run(t, new(NotTestSuite))
}
