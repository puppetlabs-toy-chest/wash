package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type NullTestSuite struct {
	PrimitiveValueTestSuite
}

func (s *NullTestSuite) TestMarshal() {
	s.MTC(Null(), nil)
}

func (s *NullTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", ".*null", true)
}

func (s *NullTestSuite) TestEvalValue() {
	s.EVFTC(nil, "foo", 1, true)
	s.EVTTC(nil, nil)
}

func (s NullTestSuite) TestEvalValueSchema() {
	s.EVSFTC(nil, s.VS("object", "array")...)
	s.EVSTTC(nil, s.VS("null")...)
}

func (s *NullTestSuite) TestExpression_AtomAndNot() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("null", true, func() rql.ASTNode {
			return Null()
		})
	}

	s.EVFTC(nil, "foo", 1, true)
	s.EVTTC(nil, nil)
	s.EVSFTC(nil, s.VS("object", "array")...)
	s.EVSTTC(nil, s.VS("null")...)
	s.AssertNotImplemented(
		nil,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.EVTTC(s.A("NOT", nil), "foo", 1, true)
	s.EVFTC(s.A("NOT", nil), nil)
	s.EVSTTC(s.A("NOT", nil), s.VS("null", "object", "array")...)
}

func TestNull(t *testing.T) {
	s := new(NullTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Null()
	}
	suite.Run(t, s)
}
