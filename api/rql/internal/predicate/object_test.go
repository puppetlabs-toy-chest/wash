package predicate

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type ObjectTestSuite struct {
	CollectionTestSuite
}

func (s *ObjectTestSuite) TestMarshal_ElementPredicate() {
	inputs := []interface{}{
		s.A("object", s.A(s.A("key", "0"), true)),
	}
	for _, input := range inputs {
		p := Object()
		s.MUM(p, input)
		s.MTC(p, input)
	}
}

func (s *ObjectTestSuite) TestUnmarshalErrors_ElementPredicate() {
	s.NodeConstructor = func() rql.ASTNode {
		return Object().(*object).collectionBase.elementPredicate
	}

	// Start by testing the match errors
	s.UMETC(s.A(), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(s.A(true), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(s.A("foo"), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(s.A(s.A("foo", "bar")), "formatted.*<element_selector>.*PE ValuePredicate", true)

	// Now test the syntax errors
	s.UMETC(s.A(s.A("key", "0", "foo"), true), "formatted.*<element_selector>.*PE ValuePredicate", false)
	s.UMETC(s.A(s.A("key"), true), "missing.*key", false)
	s.UMETC(s.A(s.A("key", float64(1)), true), "key.*string", false)
	s.UMETC(s.A(s.A("key", "foo"), true, "bar"), "formatted.*<element_selector>.*PE ValuePredicate", false)
	s.UMETC(s.A(s.A("key", "foo")), "formatted.*<element_selector>.*PE ValuePredicate.*missing.*PE ValuePredicate", false)
}

func (s *ObjectTestSuite) TestEvalValue_ElementPredicate() {
	ast := s.A("object", s.A(s.A("key", "fOo"), true))
	// Test with different keys to ensure that the object predicate finds the first matching key
	for _, key := range []string{"foo", "FOO", "foO"} {
		s.EVFTC(ast, "foo", true, []interface{}{}, map[string]interface{}{"bar": true}, map[string]interface{}{key: false})
		s.EVTTC(ast, map[string]interface{}{key: true})
	}
}

func (s *ObjectTestSuite) TestEvalValueSchema_ElementPredicate() {
	ast := s.A("object", s.A(s.A("key", "fOo"), true))
	s.EVSFTC(
		ast,
		VS{"type": "number"},
		VS{"type": "array"},
		VS{"type": "object", "properties": VS{"bar": VS{}}, "additionalProperties": false},
	)
	s.EVSTTC(ast, VS{"type": "object"})
	// Test with different keys to ensure that the object predicate finds the first matching key
	for _, key := range []string{"foo", "FOO", "foO"} {
		s.EVSTTC(ast, VS{"type": "object", "properties": VS{key: VS{}}, "additionalProperties": false})
	}
}

func (s *ObjectTestSuite) TestExpression_AtomAndNot_ElementPredicate() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("object", true, func() rql.ASTNode {
			return Object()
		})
	}

	ast := s.A("object", s.A(s.A("key", "foo"), true))
	s.EVFTC(ast, "foo", map[string]interface{}{"foo": false})
	s.EVTTC(ast, map[string]interface{}{"foo": true})
	s.EVSFTC(ast, VS{"type": "number"})
	s.EVSTTC(ast, VS{"type": "object"})

	notAST := s.A("NOT", ast)
	s.EVTTC(notAST, "foo", map[string]interface{}{"foo": false})
	s.EVFTC(notAST, map[string]interface{}{"foo": true})
	s.EVSTTC(notAST, VS{"type": "number"}, VS{"type": "object"})

	// Assert that the unmarshaled atom doesn't implement the other *Predicate
	// interfaces
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestObject(t *testing.T) {
	s := new(ObjectTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Object()
	}
	suite.Run(t, s)
}
