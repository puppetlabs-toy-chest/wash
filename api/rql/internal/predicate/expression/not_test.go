package expression

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/stretchr/testify/suite"
)

// These test Marshal/Unmarshal. Correctness of the Eval* methods
// is contained in the relevant predicate's unit tests to ensure
// that negation semantics are correct.

type NotTestSuite struct {
	asttest.Suite
}

func (s *NotTestSuite) TestMarshal() {
	s.MTC(Not(newMockP("10")), s.A("NOT", "10"))
}

func (s *NotTestSuite) TestUnmarshalErrors() {
	s.UMETC(1, `formatted.*"NOT".*<pe>`, true)
	s.UMETC(s.A("NOT", "10", "11"), `formatted.*"NOT".*<pe>`, false)
	s.UMETC(s.A("NOT"), "NOT.*expression", false)
	s.UMETC(s.A("NOT", 1), "NOT.*error.*expression.*string", false)
}

func TestNot(t *testing.T) {
	s := new(NotTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Not(&mockPtype{})
	}
	suite.Run(t, s)
}
