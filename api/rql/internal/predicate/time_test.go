package predicate

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type TimeTestSuite struct {
	asttest.Suite
}

func (s *TimeTestSuite) TestMarshal() {
	s.MTC(Time(LT, s.TM(1000)), s.A("<", s.TM(1000)))
}

func (s *TimeTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", "formatted.*<comparison_op>.*<time>", true)
	s.UMETC(s.A("foo"), "formatted.*<comparison_op>.*<time>", true)
	s.UMETC(s.A("<", "foo", "bar"), "formatted.*<comparison_op>.*<time>", false)
	s.UMETC(s.A("<"), "formatted.*<comparison_op>.*<time>.*missing.*time", false)
	s.UMETC(s.A("<", true), "valid.*time.Time.*type", false)
	s.UMETC(s.A("<", "true"), "parse.*true.*time.Time", false)
}

func (s *TimeTestSuite) TestEvalTime() {
	// Test LT
	ast := s.A("<", 1000)
	s.ETFTC(ast, s.TM(2000), s.TM(1000))
	s.ETTTC(ast, s.TM(500))

	// Test LTE
	ast = s.A("<=", 1000)
	s.ETFTC(ast, s.TM(2000))
	s.ETTTC(ast, s.TM(500), s.TM(1000))

	// Test GT
	ast = s.A(">", 1000)
	s.ETFTC(ast, s.TM(500), s.TM(1000))
	s.ETTTC(ast, s.TM(2000))

	// Test GTE
	ast = s.A(">=", 1000)
	s.ETFTC(ast, s.TM(500))
	s.ETTTC(ast, s.TM(2000), s.TM(1000))

	// Test EQL
	ast = s.A("=", 1000)
	s.ETFTC(ast, s.TM(500), s.TM(2000))
	s.ETTTC(ast, s.TM(1000))

	// Test that we can unmarshal RFC3339 times
	ast = s.A(">", s.TM(1000).Format(time.RFC3339))
	s.ETTTC(ast, s.TM(2000))
}

func (s *TimeTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("time", true, func() rql.ASTNode {
			return Time("", s.TM(0))
		})
	}

	ast := s.A("<", 1000)
	s.ETFTC(ast, s.TM(2000), s.TM(1000))
	s.ETTTC(ast, s.TM(500))
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.ETTTC(notAST, s.TM(2000), s.TM(1000))
	s.ETFTC(notAST, s.TM(500))
}

func TestTime(t *testing.T) {
	s := new(TimeTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Time("", s.TM(0))
	}
	suite.Run(t, s)
}

type TimeValueTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *TimeValueTestSuite) TestMarshal() {
	s.MTC(TimeValue(Time(LT, s.TM(1000))), s.A("time", s.A("<", s.TM(1000))))
}

func (s *TimeValueTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `formatted.*"time".*NPE TimePredicate`, true)
	s.UMETC(s.A("time", "foo", "bar"), `formatted.*"time".*NPE TimePredicate`, false)
	s.UMETC(s.A("time"), `formatted.*"time".*NPE TimePredicate.*missing.*NPE TimePredicate`, false)
	s.UMETC(s.A("time", s.A()), "formatted.*<comparison_op>.*<time>", false)
}

func (s *TimeValueTestSuite) TestEvalValue() {
	ast := s.A("time", s.A("<", 1000))
	s.EVFTC(ast, s.TM(2000), "foo")
	s.EVTTC(ast, s.TM(500), s.TM(500).Format(time.RFC3339))
	// TestEvalTime contained the operator-specific test-cases
}

func (s *TimeValueTestSuite) TestEvalValueSchema() {
	ast := s.A("time", s.A("<", 1000))
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("integer", "number", "string")...)
}

func (s *TimeValueTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("time", true, func() rql.ASTNode {
			return TimeValue(Time("", s.TM(0)))
		})
	}

	ast := s.A("time", s.A("<", 1000))
	s.EVFTC(ast, s.TM(2000), s.TM(1000))
	s.EVTTC(ast, s.TM(500))
	s.EVSFTC(ast, s.VS("object", "array")...)
	s.EVSTTC(ast, s.VS("integer", "number", "string")...)
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.EVTTC(notAST, s.TM(2000), s.TM(1000))
	s.EVFTC(notAST, s.TM(500))
	s.EVSTTC(notAST, s.VS("object", "array", "integer", "number", "string")...)
}

func TestTimeValue(t *testing.T) {
	s := new(TimeValueTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return TimeValue(Time("", s.TM(0)))
	}
	suite.Run(t, s)
}
