package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/stretchr/testify/suite"
)

// These test Marshal/Unmarshal. Correctness of the Eval* methods
// is contained in the relevant predicate's unit tests

type AtomTestSuite struct {
	asttest.Suite
}

func (s *AtomTestSuite) TestMarshal() {
	s.MTC(Atom(newMockP("10")), "10")
}

func (s *AtomTestSuite) TestUnmarshal() {
	p := Atom(&mockPtype{})
	s.UMETC(p, 1, "string", true)
	s.UMTC(p, "10", Atom(newMockP("10")))
}

func TestAtom(t *testing.T) {
	suite.Run(t, new(AtomTestSuite))
}
