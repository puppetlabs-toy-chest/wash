package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/stretchr/testify/suite"
)

// These test Marshal/Unmarshal. Correctness of the Eval* methods
// is contained in the relevant predicate's unit tests

type AtomTestSuite struct {
	asttest.Suite
}

func (s *AtomTestSuite) TestMarshal() {
	s.MTC(Atom(predicate.Boolean(true)), predicate.Boolean(true).Marshal())
}

func (s *AtomTestSuite) TestUnmarshal() {
	p := Atom(predicate.Boolean(false))
	s.UMETC(p, "foo", "Boolean", true)
	s.UMTC(p, true, Atom(predicate.Boolean(true)))
}

func TestAtom(t *testing.T) {
	suite.Run(t, new(AtomTestSuite))
}
