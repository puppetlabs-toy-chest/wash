package predicate

import (
	"fmt"
	"strconv"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
)

// Contains common test code for Object/Array predicates

type CollectionTestSuite struct {
	asttest.Suite
	isArray bool
}

// Saves some clutter when creating nested schemas
type VS = map[string]interface{}

func (s *CollectionTestSuite) TestMarshal_SizePredicate() {
	n := s.NodeConstructor()
	s.MUM(n, s.A(s.ctype(), s.A("size", s.A("<", "10"))))
	s.MTC(n, s.A(s.ctype(), s.A("size", s.A("<", "10"))))
}

func (s *CollectionTestSuite) TestUnmarshalErrors() {
	fmtErrMsg := fmt.Sprintf("formatted.*%v.*<size_predicate>.*<%v_element_predicate>", s.ctype(), s.ctype())
	s.UMETC("foo", fmtErrMsg, true)
	s.UMETC(s.A(s.ctype(), s.A("size", s.A("<", "10")), s.A("size", s.A("<", "10"))), fmtErrMsg, false)
	s.UMETC(s.A(s.ctype()), fmt.Sprintf("%v.*missing.*predicate", fmtErrMsg), false)
	s.UMETC(s.A(s.ctype(), s.A()), fmt.Sprintf("error.*unmarshalling.*%v.*expected", s.ctype()), false)
	s.UMETC(s.A(s.ctype(), s.A("size")), fmt.Sprintf("error.*unmarshalling.*%v.*size", s.ctype()), false)
	var selector interface{}
	if s.isArray {
		selector = "some"
	} else {
		selector = []interface{}{"key", "0"}
	}
	s.UMETC(s.A(s.ctype(), s.A(selector)), "formatted.*<element_selector>.*NPE ValuePredicate.*missing.*NPE ValuePredicate", false)
}

func (s *CollectionTestSuite) TestEvalValue_SizePredicate() {
	ast := s.A(s.ctype(), s.A("size", s.A(">", "0")))
	s.EVFTC(ast, "foo", true, s.ISPV(), s.SPV(0))
	s.EVTTC(ast, s.SPV(1))
}

func (s *CollectionTestSuite) TestEvalValueSchema_SizePredicate() {
	ast := s.A(s.ctype(), s.A("size", s.A(">", "0")))
	s.EVSFTC(ast, VS{"type": "number"}, s.ISPVS())
	s.EVSTTC(ast, s.SPVS())
}

func (s *CollectionTestSuite) TestExpression_AtomAndNot_SizePredicate() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New(s.ctype(), true, func() rql.ASTNode {
			return s.DefaultNodeConstructor()
		})
	}

	ast := s.A(s.ctype(), s.A("size", s.A(">", "0")))
	s.EVFTC(ast, s.SPV(0))
	s.EVTTC(ast, s.SPV(1))
	s.EVSFTC(ast, VS{"type": "number"})
	s.EVSTTC(ast, s.SPVS())
	s.AssertNotImplemented(
		ast,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	notAST := s.A("NOT", ast)
	s.EVTTC(notAST, s.SPV(0))
	s.EVFTC(notAST, s.SPV(1))
	s.EVSTTC(notAST, VS{"type": "number"}, s.SPVS(), s.ISPVS())
}

func (s *CollectionTestSuite) TestSizePredicate_AcceptsNumericPEs() {
	// rtc => runTestCase
	rtc := func(expr interface{}, trueV int) {
		s.EVTTC(s.A(s.ctype(), s.A("size", expr)), s.SPV(trueV))
	}

	rtc(s.A(">", float64(500)), 1000)
	rtc(s.A("NOT", s.A(">", float64(500))), 500)
	rtc(s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500))), 500)
	rtc(s.A("OR", s.A(">", float64(500)), s.A("=", float64(500))), 500)
}

func (s *CollectionTestSuite) TestElementPredicate_AcceptsValueNPEs() {
	// rtc => runTestCase
	rtc := func(expr interface{}, trueV interface{}) {
		s.EVTTC(s.A(s.ctype(), s.A(s.selector(), expr)), s.EPV(trueV))
	}
	// timeV => timeValue
	timeV := func(unixSeconds int64) string {
		return time.Unix(unixSeconds, 0).Format(time.RFC3339)
	}

	// Test that it unmarshals each of the atoms, including their corresponding
	// NPEs
	rtc(s.A("object", s.A("size", s.A(">", "0"))), map[string]interface{}{"0": nil})
	rtc(s.A("array", s.A("size", s.A(">", "0"))), []interface{}{true})
	rtc(nil, nil)
	rtc(true, true)
	// Test "number"
	rtc(s.A("number", s.A(">", float64(500))), float64(1000))
	rtc(s.A("number", s.A("NOT", s.A(">", float64(500)))), float64(500))
	rtc(s.A("number", s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500)))), float64(500))
	rtc(s.A("number", s.A("OR", s.A(">", float64(500)), s.A("=", float64(500)))), float64(500))
	// Test "time"
	rtc(s.A("time", s.A(">", float64(500))), timeV(1000))
	rtc(s.A("time", s.A("NOT", s.A(">", float64(500)))), timeV(500))
	rtc(s.A("time", s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500)))), timeV(500))
	rtc(s.A("time", s.A("OR", s.A(">", float64(500)), s.A("=", float64(500)))), timeV(500))
	// Test "string"
	rtc(s.A("string", s.A("glob", "foo")), "foo")
	rtc(s.A("string", s.A("regex", "foo")), "foo")
	rtc(s.A("string", s.A("=", "foo")), "foo")
	rtc(s.A("string", s.A("NOT", s.A("glob", "foo"))), "bar")
	rtc(s.A("string", s.A("AND", s.A("glob", "*o*"), s.A("glob", "foo"))), "foo")
	rtc(s.A("string", s.A("OR", s.A("glob", "foo"), s.A("glob", "bar"))), "bar")

	// Now test that it can unmarshal the operators
	rtc(s.A("NOT", true), false)
	rtc(s.A("AND", true, true), true)
	rtc(s.A("OR", false, true), true)
}

func (s *CollectionTestSuite) TestElementPredicate_EvalValueSchema_NestedNPEs() {
	rtc := func(expr interface{}, trueVS VS, falseVS VS) {
		ast := s.A(s.ctype(), s.A(s.selector(), expr))
		s.EVSTTC(ast, s.mergeVS(trueVS))
		s.EVSFTC(ast, s.mergeVS(falseVS))
	}

	// Test single-level nesting
	rtc(
		s.A("object", s.A(s.A("key", "foo"), nil)),
		VS{"type": "object", "additionalProperties": false, "properties": VS{"foo": VS{"type": "number"}}},
		VS{"type": "object", "additionalProperties": false, "properties": VS{"foo": VS{"type": "object"}}},
	)
	rtc(
		s.A("array", s.A("some", nil)),
		VS{"type": "array", "items": VS{"type": "number"}},
		VS{"type": "array", "items": VS{"type": "object"}},
	)

	// Test multi-level nesting
	rtc(
		s.A("object", s.A(s.A("key", "foo"), s.A("array", s.A("some", nil)))),
		VS{
			"type":                 "object",
			"additionalProperties": false,
			"properties": VS{
				"foo": VS{
					"type":  "array",
					"items": VS{"type": "number"},
				},
			},
		},
		VS{
			"type":                 "object",
			"additionalProperties": false,
			"properties": VS{
				// bar is not a valid key so this should return false
				"bar": VS{
					"type":  "array",
					"items": VS{"type": "number"},
				},
			},
		},
	)
	rtc(
		s.A("array", s.A("some", s.A("object", s.A(s.A("key", "foo"), nil)))),
		VS{
			"type": "array",
			"items": VS{
				"type": "object",
				"properties": VS{
					"foo": VS{
						"type": "number",
					},
				},
			},
		},
		VS{
			"type": "array",
			"items": VS{
				"type": "number",
			},
		},
	)
}

func (s *CollectionTestSuite) ctype() string {
	if s.isArray {
		return "array"
	} else {
		return "object"
	}
}

func (s *CollectionTestSuite) selector() interface{} {
	if s.isArray {
		return "some"
	} else {
		return []interface{}{"key", "0"}
	}
}

func (s *CollectionTestSuite) mergeVS(childVS VS) VS {
	if s.isArray {
		return VS{"type": "array", "items": childVS}
	} else {
		return VS{"type": "object", "properties": VS{"0": childVS}, "additionalProperties": false}
	}
}

// SPV => SizePredicateValue
func (s *CollectionTestSuite) SPV(numElem int) interface{} {
	if s.isArray {
		arrayV := []interface{}{}
		for i := 0; i < numElem; i++ {
			arrayV = append(arrayV, strconv.Itoa(i))
		}
		return arrayV
	} else {
		objectV := make(map[string]interface{})
		for i := 0; i < numElem; i++ {
			objectV[strconv.Itoa(i)] = nil
		}
		return objectV
	}
}

// ISPV => InvalidSizePredicateValue
func (s *CollectionTestSuite) ISPV() interface{} {
	if s.isArray {
		return map[string]interface{}{}
	} else {
		return []interface{}{}
	}
}

// SPVS => SizePredicateValueSchema
func (s *CollectionTestSuite) SPVS() VS {
	return VS{"type": s.ctype()}
}

// ISPVS => InvalidSizePredicateValueSchema
func (s *CollectionTestSuite) ISPVS() VS {
	if s.isArray {
		return VS{"type": "object"}
	} else {
		return VS{"type": "array"}
	}
}

// EPV => ElementPredicateValue
func (s *CollectionTestSuite) EPV(elem interface{}) interface{} {
	if s.isArray {
		return []interface{}{elem}
	} else {
		return map[string]interface{}{"0": elem}
	}
}
