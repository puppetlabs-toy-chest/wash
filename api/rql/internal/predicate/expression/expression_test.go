package expression

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
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
	s.UMETC(e, 1, "expected.*PE.*mock.*predicate", true)
	s.UMETC(e, "syntax", "failed.*unmarshal.*PE.*mock.*predicate.*syntax.*error", false)
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

// mockPtype is a mock predicate type used to test the top-level
// expression class and each of the binary operators. Each of the
// Eval* methods "serialize" the specific type into something
// that can be compared with "v".
type mockPtype struct {
	v string
}

func newMockP(v string) *mockPtype {
	return &mockPtype{v}
}

func (p *mockPtype) Marshal() interface{} {
	return p.v
}

func (p *mockPtype) Unmarshal(input interface{}) error {
	str, ok := input.(string)
	if !ok {
		return errz.MatchErrorf("expected a string value")
	}
	if str == "syntax" {
		return fmt.Errorf("syntax error")
	}
	p.v = str
	return nil
}

func (p *mockPtype) EntryInDomain(rql.Entry) bool {
	return true
}

func (p *mockPtype) EvalEntry(e rql.Entry) bool {
	return e.Name == p.v
}

func (p *mockPtype) EntrySchemaInDomain(*rql.EntrySchema) bool {
	return true
}

func (p *mockPtype) EvalEntrySchema(s *rql.EntrySchema) bool {
	return s.Path() == p.v
}

func (p *mockPtype) ValueInDomain(interface{}) bool {
	return true
}

func (p *mockPtype) EvalValue(v interface{}) bool {
	return v == p.v
}

func (p *mockPtype) EvalString(str string) bool {
	return str == p.v
}

func (p *mockPtype) EvalNumeric(x decimal.Decimal) bool {
	return x.String() == p.v
}

func (p *mockPtype) EvalTime(t time.Time) bool {
	return strconv.Itoa(int(t.Unix())) == p.v
}

func (p *mockPtype) EvalAction(action plugin.Action) bool {
	return p.v == action.Name
}

var _ = rql.EntryPredicate(&mockPtype{})
var _ = rql.EntrySchemaPredicate(&mockPtype{})
var _ = rql.ValuePredicate(&mockPtype{})
var _ = rql.StringPredicate(&mockPtype{})
var _ = rql.NumericPredicate(&mockPtype{})
var _ = rql.TimePredicate(&mockPtype{})
var _ = rql.ActionPredicate(&mockPtype{})
