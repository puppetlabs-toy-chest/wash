package expression

import (
	"fmt"
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/stretchr/testify/suite"
)

/*
These tests are meant to test that we can unmarshal a PE
to its reduced version (and that we correctly return any
errors). Testing the correctness of the Eval* methods
themselves are left to the places that use the expression
type
*/

type ExpressionTestSuite struct {
	asttest.Suite
}

func (s *ExpressionTestSuite) UMTC(input interface{}, expected interface{}) {
	e := s.mockExpression().(*expression)
	if s.NoError(e.Unmarshal(input)) {
		s.Equal(expected, e.reducedForm.Marshal())
	}
}

func (s *ExpressionTestSuite) TestUnmarshal_Errors() {
	e := s.mockExpression()
	s.UMETC(e, "bar", "expected.*PE.*mock.*predicate", true)
	s.UMETC(e, "foo", "failed.*unmarshal.*PE.*mock.*predicate.*syntax.*error", false)
}

func (s *ExpressionTestSuite) TestUnmarshal() {
	// Test simple unmarshaling (Atom, Not, Binop)
	s.UMTC("p", "p")
	s.UMTC(s.A("NOT", "p"), s.A("NOT", "p"))
	s.UMTC(s.A("AND", "p", "p"), s.A("AND", "p", "p"))
	s.UMTC(s.A("OR", "p", "p"), s.A("OR", "p", "p"))

	// Test nested unmarshaling
	s.UMTC(s.A("AND", s.A("NOT", "p"), s.A("OR", "p", "p")), s.A("AND", s.A("NOT", "p"), s.A("OR", "p", "p")))
	s.UMTC(s.A("OR", s.A("NOT", "p"), s.A("AND", "p", "p")), s.A("OR", s.A("NOT", "p"), s.A("AND", "p", "p")))

	// Test NOT reductions
	//
	// NOT(NOT(p)) == p
	s.UMTC(s.A("NOT", s.A("NOT", "p")), "p")
	// NOT(AND(p, p)) == OR(NOT(p), NOT(p))
	s.UMTC(s.A("NOT", s.A("AND", "p", "p")), s.A("OR", s.A("NOT", "p"), s.A("NOT", "p")))
	// NOT(OR(p, p)) == AND(NOT(p), NOT(p))
	s.UMTC(s.A("NOT", s.A("OR", "p", "p")), s.A("AND", s.A("NOT", "p"), s.A("NOT", "p")))

	// Test a more complicated reduction
	//
	// AND(NOT(OR(p, NOT(p))), OR(NOT(AND(NOT(p), p)), NOT(p))) ==
	// AND(AND(NOT(p), p), OR(OR(p, NOT(p)), NOT(p)))
	s.UMTC(
		s.A("AND", s.A("NOT", s.A("OR", "p", s.A("NOT", "p"))), s.A("OR", s.A("NOT", s.A("AND", s.A("NOT", "p"), "p")), s.A("NOT", "p"))),
		s.A("AND", s.A("AND", s.A("NOT", "p"), "p"), s.A("OR", s.A("OR", "p", s.A("NOT", "p")), s.A("NOT", "p"))),
	)
}

func TestExpression(t *testing.T) {
	suite.Run(t, new(ExpressionTestSuite))
}

func (s *ExpressionTestSuite) mockExpression() rql.ASTNode {
	return New("mock predicate", func() rql.ASTNode { return &mockPtype{} })
}

type mockPtype struct{}

func (p *mockPtype) Marshal() interface{} {
	return "p"
}

func (p *mockPtype) Unmarshal(input interface{}) error {
	if input != "p" {
		if input == "foo" {
			// Mock a syntax error
			return fmt.Errorf("syntax error")
		}
		return errz.MatchErrorf("expected 'p', got %v", input)
	}
	return nil
}

var _ = rql.ASTNode(&mockPtype{})
