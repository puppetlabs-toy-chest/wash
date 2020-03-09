package expression

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/api/rql/internal/primary/meta"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/suite"
)

/*
These tests are meant to test that we can unmarshal a PE/NPE
(and that we correctly return any errors). Testing the correctness
of the Eval* methods themselves are left to the places that use the
expression type
*/

type ExpressionTestSuite struct {
	asttest.Suite
}

func (s *ExpressionTestSuite) TestUnmarshal_PE_Errors() {
	s.NodeConstructor = func() rql.ASTNode {
		return s.mockExpression(false)
	}
	s.UMETC(1, "expected.*PE.*mock.*predicate", true)
	s.UMETC(s.A("NOT", "p"), "expected.*PE.*mock.*predicate", true)
	s.UMETC("syntax", "failed.*unmarshal.*PE.*mock.*predicate.*syntax.*error", false)

	// Here we test that the operators fail to unmarshal "NOT"
	for _, op := range []string{"AND", "OR"} {
		s.UMETC(s.A(op, s.A("NOT", "p"), "p"), "error.*LHS.*PE", false)
		s.UMETC(s.A(op, "p", s.A("NOT", "p")), "error.*RHS.*PE", false)
	}
}

func (s *ExpressionTestSuite) TestUnmarshal_PE() {
	UMTC := func(input interface{}, expected interface{}) {
		e := s.mockExpression(false)
		if s.NoError(e.Unmarshal(input)) {
			s.Equal(expected, e.Marshal())
		}
	}

	// Test simple unmarshaling (Atom, Binop)
	UMTC("p", "p")
	UMTC(s.A("AND", "p", "p"), s.A("AND", "p", "p"))
	UMTC(s.A("OR", "p", "p"), s.A("OR", "p", "p"))

	// Test nested unmarshaling
	UMTC(s.A("AND", "p", s.A("OR", "p", "p")), s.A("AND", "p", s.A("OR", "p", "p")))
	UMTC(s.A("OR", "p", s.A("AND", "p", "p")), s.A("OR", "p", s.A("AND", "p", "p")))
}

func (s *ExpressionTestSuite) TestUnmarshal_NPE_Errors() {
	s.NodeConstructor = func() rql.ASTNode {
		return s.mockExpression(true)
	}
	s.UMETC(1, "expected.*NPE.*mock.*predicate", true)
	s.UMETC("syntax", "failed.*unmarshal.*NPE.*mock.*predicate.*syntax.*error", false)
}

func (s *ExpressionTestSuite) TestUnmarshal_NPE() {
	UMTC := func(input interface{}, expected interface{}) {
		e := s.mockExpression(true)
		if s.NoError(e.Unmarshal(input)) {
			s.Equal(expected, e.Marshal())
		}
	}

	// Test simple unmarshaling (Atom, Not, Binop)
	UMTC("p", "p")
	UMTC(s.A("NOT", "p"), s.A("NOT", "p"))
	UMTC(s.A("AND", "p", "p"), s.A("AND", "p", "p"))
	UMTC(s.A("OR", "p", "p"), s.A("OR", "p", "p"))

	// Test nested unmarshaling
	UMTC(s.A("AND", s.A("NOT", "p"), s.A("OR", "p", "p")), s.A("AND", s.A("NOT", "p"), s.A("OR", "p", "p")))
	UMTC(s.A("OR", s.A("NOT", "p"), s.A("AND", "p", "p")), s.A("OR", s.A("NOT", "p"), s.A("AND", "p", "p")))
}

func (s *ExpressionTestSuite) TestReduceNPE() {
	rtc := func(input interface{}, expected interface{}) {
		e := s.mockExpression(true)
		if s.NoError(e.Unmarshal(input)) {
			reducedForm := reduce(e.MatchedNode())
			s.Equal(expected, reducedForm.Marshal())
		}
	}

	// Test NOT reductions
	//
	// NOT(NOT(p)) == p
	rtc(s.A("NOT", s.A("NOT", "p")), "p")

	// NOT(AND(p, p)) == OR(NOT(p), NOT(p))
	rtc(s.A("NOT", s.A("AND", "p", "p")), s.A("OR", s.A("NOT", "p"), s.A("NOT", "p")))
	// NOT(OR(p, p)) == AND(NOT(p), NOT(p))
	rtc(s.A("NOT", s.A("OR", "p", "p")), s.A("AND", s.A("NOT", "p"), s.A("NOT", "p")))

	// Test a more complicated reduction
	//
	// AND(NOT(OR(p, NOT(p))), OR(NOT(AND(NOT(p), p)), NOT(p))) ==
	// AND(AND(NOT(p), p), OR(OR(p, NOT(p)), NOT(p)))
	rtc(
		s.A("AND", s.A("NOT", s.A("OR", "p", s.A("NOT", "p"))), s.A("OR", s.A("NOT", s.A("AND", s.A("NOT", "p"), "p")), s.A("NOT", "p"))),
		s.A("AND", s.A("AND", s.A("NOT", "p"), "p"), s.A("OR", s.A("OR", "p", s.A("NOT", "p")), s.A("NOT", "p"))),
	)
}

func TestExpression(t *testing.T) {
	suite.Run(t, new(ExpressionTestSuite))
}

func (s *ExpressionTestSuite) mockExpression(negatable bool) *expression {
	return New("mock predicate", negatable, func() rql.ASTNode { return &mockPtype{} }).(*expression)
}

// mockPtype is a mock predicate type used to test the top-level
// expression class and each of the binary operators. Each of the
// Eval* methods "serialize" the specific type into something
// that can be compared with "v".
type mockPtype struct {
	*meta.ValuePredicateBase
	v string
}

func newMockP(v string) *mockPtype {
	p := &mockPtype{v: v}
	p.ValuePredicateBase = meta.NewValuePredicate(p)
	return p
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

func (p *mockPtype) IsPrimary() bool {
	return true
}

func (p *mockPtype) EvalEntry(e rql.Entry) bool {
	return e.Name == p.v
}

func (p *mockPtype) EvalEntrySchema(s *rql.EntrySchema) bool {
	return s.Path() == p.v
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

func (p *mockPtype) SchemaPredicate(svs meta.SatisfyingValueSchema) meta.SchemaPredicate {
	return meta.MakeSchemaPredicate(svs.AddObject(p.v).EndsWithPrimitiveValue())
}

var _ = rql.EntryPredicate(&mockPtype{})
var _ = rql.EntrySchemaPredicate(&mockPtype{})
var _ = meta.ValuePredicate(&mockPtype{})
var _ = rql.StringPredicate(&mockPtype{})
var _ = rql.NumericPredicate(&mockPtype{})
var _ = rql.TimePredicate(&mockPtype{})
var _ = rql.ActionPredicate(&mockPtype{})
