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
		s.A("object", s.A(s.A("key", "0"), s.A("boolean", true))),
	}
	for _, input := range inputs {
		p := Object()
		s.MUM(p, input)
		s.MTC(p, input)
	}
}

func (s *ObjectTestSuite) TestUnmarshalErrors_ElementPredicate() {
	// Start by testing the match errors
	p := Object().(*object).collectionBase.elementPredicate
	s.UMETC(p, s.A(), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(p, s.A(true), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(p, s.A("foo"), "formatted.*<element_selector>.*PE ValuePredicate", true)
	s.UMETC(p, s.A(s.A("foo", "bar")), "formatted.*<element_selector>.*PE ValuePredicate", true)

	// Now test the syntax errors
	s.UMETC(p, s.A(s.A("key", "0", "foo"), true), "formatted.*<element_selector>.*PE ValuePredicate", false)
	s.UMETC(p, s.A(s.A("key"), true), "missing.*key", false)
	s.UMETC(p, s.A(s.A("key", float64(1)), true), "key.*string", false)
	s.UMETC(p, s.A(s.A("key", "foo"), true, "bar"), "formatted.*<element_selector>.*PE ValuePredicate", false)
	s.UMETC(p, s.A(s.A("key", "foo")), "formatted.*<element_selector>.*PE ValuePredicate.*missing.*PE ValuePredicate", false)
}

func (s *ObjectTestSuite) TestEvalValue_ElementPredicate() {
	p := Object()
	s.MUM(p, s.A("object", s.A(s.A("key", "fOo"), s.A("boolean", true))))
	// Test with different keys to ensure that the object predicate finds the first matching key
	for _, key := range []string{"foo", "FOO", "foO"} {
		s.EVFTC(p, "foo", true, []interface{}{}, map[string]interface{}{"bar": true}, map[string]interface{}{key: false})
		s.EVTTC(p, map[string]interface{}{key: true})
	}
}

func (s *ObjectTestSuite) TestExpression_AtomAndNot_ElementPredicate() {
	expr := expression.New("object", true, func() rql.ASTNode {
		return Object()
	})

	s.MUM(expr, s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true))))
	s.EVFTC(expr, "foo", map[string]interface{}{"foo": false})
	s.EVTTC(expr, map[string]interface{}{"foo": true})
	s.MUM(expr, s.A("NOT", s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true)))))
	s.EVTTC(expr, "foo", map[string]interface{}{"foo": false})
	s.EVFTC(expr, map[string]interface{}{"foo": true})

	// Assert that the unmarshaled atom doesn't implement the other *Predicate
	// interfaces
	s.MUM(expr, s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true))))
	s.AssertNotImplemented(
		expr,
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
	suite.Run(t, s)
}
