package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type ArrayTestSuite struct {
	CollectionTestSuite
}

func (s *ArrayTestSuite) TestMarshal_ElementPredicate() {
	inputs := []interface{}{
		s.A("array", s.A("some", true)),
		s.A("array", s.A("all", true)),
		s.A("array", s.A(float64(0), true)),
	}
	for _, input := range inputs {
		p := Array()
		s.MUM(p, input)
		s.MTC(p, input)
	}
}

func (s *ArrayTestSuite) TestUnmarshalErrors_ElementPredicate() {
	s.NodeConstructor = func() rql.ASTNode {
		return Array().(*array).collectionBase.elementPredicate
	}

	// Start by testing the match errors
	s.UMETC(s.A(), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(s.A(true), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(s.A("foo"), "formatted.*<element_selector>.*PE ValuePredicate", true)

	// Now test the syntax errors
	selectors := []interface{}{
		"some",
		"all",
		float64(1),
	}
	for _, selector := range selectors {
		s.UMETC(s.A(selector, s.A("string", s.A("=", "foo")), "bar"), "formatted.*<element_selector>.*PE ValuePredicate", false)
		s.UMETC(s.A(selector), "formatted.*<element_selector>.*PE ValuePredicate.*missing.*PE ValuePredicate", false)
	}
	s.UMETC(s.A(float64(-10), "foo"), "array.*index.*unsigned.*int", false)
}

func (s *ArrayTestSuite) TestEvalValue_ElementPredicate() {
	// Test "some"
	ast := s.A("array", s.A("some", true))
	s.EVFTC(ast, "foo", true, []interface{}{false}, []interface{}{})
	s.EVTTC(ast, []interface{}{true}, []interface{}{false, true})

	// Test "all"
	ast = s.A("array", s.A("all", true))
	s.EVFTC(ast, "foo", true, []interface{}{false}, []interface{}{true, false})
	s.EVTTC(ast, []interface{}{true}, []interface{}{true, true})

	// Test "n"
	ast = s.A("array", s.A(float64(0), true))
	s.EVFTC(ast, "foo", true, []interface{}{"foo", "bar"}, []interface{}{false, true})
	s.EVTTC(ast, []interface{}{true}, []interface{}{true, "foo"})
	// Add a case with a non-empty array
	ast = s.A("array", s.A(float64(1), true))
	s.EVFTC(ast, "foo", true, []interface{}{true, false})
	s.EVTTC(ast, []interface{}{false, true}, []interface{}{"foo", true})
}

func (s *ArrayTestSuite) TestEvalValueSchema_ElementPredicate() {
	for _, selector := range []interface{}{"some", "all", float64(0)} {
		ast := s.A("array", s.A(selector, true))
		s.EVSFTC(ast, VS{"type": "number"}, VS{"type": "object"}, VS{"type": "array", "items": VS{"type": "object"}})
		s.EVSTTC(ast, VS{"type": "array"}, VS{"type": "array", "items": VS{"type": "boolean"}})
	}
}

func (s *ArrayTestSuite) TestExpression_AtomAndNot_ElementPredicate() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("array", true, func() rql.ASTNode {
			return Array()
		})
	}

	for _, selector := range []interface{}{"some", "all", float64(0)} {
		ast := s.A("array", s.A(selector, true))
		s.EVFTC(ast, []interface{}{false})
		s.EVTTC(ast, []interface{}{true})
		s.EVSFTC(ast, VS{"type": "object"})
		s.EVSTTC(ast, VS{"type": "array"})

		notAST := s.A("NOT", ast)
		s.EVTTC(notAST, []interface{}{false})
		s.EVFTC(notAST, []interface{}{true})
		s.EVSTTC(notAST, VS{"type": "array"}, VS{"type": "object"})
	}

	// Assert that the unmarshaled atom doesn't implement the other *Predicate
	// interfaces
	s.AssertNotImplemented(
		s.A("array", s.A("some", true)),
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestArray(t *testing.T) {
	s := new(ArrayTestSuite)
	s.isArray = true
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Array()
	}
	suite.Run(t, s)
}
