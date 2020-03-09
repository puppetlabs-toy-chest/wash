package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
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

func (s *AtomTestSuite) TestUnmarshalErrors() {
	s.UMETC(1, "string", true)
}

func TestAtom(t *testing.T) {
	s := new(AtomTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Atom(newMockP(""))
	}
	suite.Run(t, s)
}
