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

func (s *TimeTestSuite) TestTime_Marshal() {
	s.MTC(Time(LT, s.TM(1000)), s.A("<", s.TM(1000)))
}

func (s *TimeTestSuite) TestTime_Unmarshal() {
	t := Time("", s.TM(0))
	s.UMETC(t, "foo", "formatted.*<comparison_op>.*<time>", true)
	s.UMETC(t, s.A("foo"), "formatted.*<comparison_op>.*<time>", true)
	s.UMETC(t, s.A("<", "foo", "bar"), "formatted.*<comparison_op>.*<time>", false)
	s.UMETC(t, s.A("<"), "formatted.*<comparison_op>.*<time>.*missing.*time", false)
	s.UMETC(t, s.A("<", true), "valid.*time.Time.*type", false)
	s.UMETC(t, s.A("<", "true"), "parse.*true.*time.Time", false)
	s.UMTC(t, s.A("<", s.TM(1000)), Time(LT, s.TM(1000)))
	rfc3339Str := s.TM(1000).Format(time.RFC3339)
	expectedTime, err := time.Parse(time.RFC3339, rfc3339Str)
	if s.NoError(err) {
		s.UMTC(t, s.A("<", rfc3339Str), Time(LT, expectedTime))
	}
}

func (s *TimeTestSuite) TestTime_EvalTime() {
	// Test LT
	t := Time(LT, s.TM(1000))
	s.ETFTC(t, s.TM(2000), s.TM(1000))
	s.ETTTC(t, s.TM(500))

	// Test LTE
	t = Time(LTE, s.TM(1000))
	s.ETFTC(t, s.TM(2000))
	s.ETTTC(t, s.TM(500), s.TM(1000))

	// Test GT
	t = Time(GT, s.TM(1000))
	s.ETFTC(t, s.TM(500), s.TM(1000))
	s.ETTTC(t, s.TM(2000))

	// Test GTE
	t = Time(GTE, s.TM(1000))
	s.ETFTC(t, s.TM(500))
	s.ETTTC(t, s.TM(2000), s.TM(1000))

	// Test EQL
	t = Time(EQL, s.TM(1000))
	s.ETFTC(t, s.TM(500), s.TM(2000))
	s.ETTTC(t, s.TM(1000))
}

func (s *TimeTestSuite) TestTime_Expression_AtomAndNot() {
	expr := expression.New("time", true, func() rql.ASTNode {
		return Time("", s.TM(0))
	})

	s.MUM(expr, []interface{}{"<", float64(1000)})
	s.ETFTC(expr, s.TM(2000), s.TM(1000))
	s.ETTTC(expr, s.TM(500))
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"<", float64(1000)}})
	s.ETTTC(expr, s.TM(2000), s.TM(1000))
	s.ETFTC(expr, s.TM(500))
}

func (s *TimeTestSuite) TestTimeValue_Marshal() {
	s.MTC(TimeValue(LT, s.TM(1000)), s.A("time", s.A("<", s.TM(1000))))
}

func (s *TimeTestSuite) TestTimeValue_Unmarshal() {
	t := TimeValue("", s.TM(0))
	s.UMETC(t, "foo", `formatted.*"time".*<time_predicate>`, true)
	s.UMETC(t, s.A("time", "foo", "bar"), `formatted.*"time".*<time_predicate>`, false)
	s.UMETC(t, s.A("time"), `formatted.*"time".*<time_predicate>.*missing.*time.*predicate`, false)
	s.UMETC(t, s.A("time", s.A()), "formatted.*<comparison_op>.*<time>", false)
	s.UMTC(t, s.A("time", s.A("<", s.TM(1000))), TimeValue(LT, s.TM(1000)))
}

func (s *TimeTestSuite) TestTimeValue_EvalValue() {
	t := TimeValue(LT, s.TM(1000))
	s.EVFTC(t, s.TM(2000))
	s.EVTTC(t, s.TM(500), s.TM(500).Format(time.RFC3339))
	// TestEvalTime contained the operator-specific test-cases
}

func (s *TimeTestSuite) TestTimeValue_Expression_AtomAndNot() {
	expr := expression.New("time", true, func() rql.ASTNode {
		return TimeValue("", s.TM(0))
	})

	s.MUM(expr, []interface{}{"time", []interface{}{"<", float64(1000)}})
	s.EVFTC(expr, s.TM(2000), s.TM(1000), "foo")
	s.EVTTC(expr, s.TM(500))
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, []interface{}{"NOT", []interface{}{"time", []interface{}{"<", float64(1000)}}})
	s.EVTTC(expr, s.TM(2000), s.TM(1000))
	s.EVFTC(expr, s.TM(500))
}

func TestTime(t *testing.T) {
	suite.Run(t, new(TimeTestSuite))
}
