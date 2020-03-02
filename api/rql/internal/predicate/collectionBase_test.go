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

func (s *CollectionTestSuite) TestMarshal_SizePredicate() {
	p := s.P()
	s.MUM(p, s.A(s.ctype(), s.A("size", s.A("<", "10"))))
	s.MTC(p, s.A(s.ctype(), s.A("size", s.A("<", "10"))))
}

func (s *CollectionTestSuite) TestUnmarshalErrors() {
	p := s.P()
	fmtErrMsg := fmt.Sprintf("formatted.*%v.*<size_predicate>.*<%v_element_predicate>", s.ctype(), s.ctype())
	s.UMETC(p, "foo", fmtErrMsg, true)
	s.UMETC(p, s.A(s.ctype(), s.A("size", s.A("<", "10")), s.A("size", s.A("<", "10"))), fmtErrMsg, false)
	s.UMETC(p, s.A(s.ctype()), fmt.Sprintf("%v.*missing.*predicate", fmtErrMsg), false)
	s.UMETC(p, s.A(s.ctype(), s.A()), fmt.Sprintf("error.*unmarshalling.*%v.*expected", s.ctype()), false)
	s.UMETC(p, s.A(s.ctype(), s.A("size")), fmt.Sprintf("error.*unmarshalling.*%v.*size", s.ctype()), false)
	var selector interface{}
	if s.isArray {
		selector = "some"
	} else {
		selector = []interface{}{"key", "0"}
	}
	s.UMETC(p, s.A(s.ctype(), s.A(selector)), "formatted.*<element_selector>.*NPE ValuePredicate.*missing.*NPE ValuePredicate", false)
}

func (s *CollectionTestSuite) TestEvalValue_SizePredicate() {
	p := s.P()
	s.MUM(p, s.A(s.ctype(), s.A("size", s.A(">", "0"))))
	s.EVFTC(p, "foo", true, s.ISPV(), s.SPV(0))
	s.EVTTC(p, s.SPV(1))
}

func (s *CollectionTestSuite) TestExpression_AtomAndNot_SizePredicate() {
	expr := expression.New(s.ctype(), true, func() rql.ASTNode {
		return s.P()
	})

	s.MUM(expr, s.A(s.ctype(), s.A("size", s.A(">", "0"))))
	s.EVFTC(expr, s.SPV(0))
	s.EVTTC(expr, s.SPV(1))
	s.AssertNotImplemented(
		expr,
		asttest.EntryPredicateC,
		asttest.EntrySchemaPredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)

	s.MUM(expr, s.A("NOT", s.A(s.ctype(), s.A("size", s.A(">", "0")))))
	s.EVTTC(expr, s.SPV(0))
	s.EVFTC(expr, s.SPV(1))
}

func (s *CollectionTestSuite) TestSizePredicate_AcceptsNumericPEs() {
	// rtc => runTestCase
	rtc := func(expr interface{}, trueV int) {
		p := s.P()
		s.MUM(p, s.A(s.ctype(), s.A("size", expr)))
		s.EVTTC(p, s.SPV(trueV))
	}

	rtc(s.A(">", float64(500)), 1000)
	rtc(s.A("NOT", s.A(">", float64(500))), 500)
	rtc(s.A("AND", s.A(">=", float64(500)), s.A("=", float64(500))), 500)
	rtc(s.A("OR", s.A(">", float64(500)), s.A("=", float64(500))), 500)
}

func (s *CollectionTestSuite) TestElementPredicate_AcceptsValueNPEs() {
	// rtc => runTestCase
	rtc := func(expr interface{}, trueV interface{}) {
		p := s.P()
		var selector interface{}
		if s.isArray {
			selector = "some"
		} else {
			selector = []interface{}{"key", "0"}
		}
		s.MUM(p, s.A(s.ctype(), s.A(selector, expr)))
		s.EVTTC(p, s.EPV(trueV))
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
	rtc(s.A("boolean", true), true)
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
	rtc(s.A("NOT", s.A("boolean", true)), false)
	rtc(s.A("AND", s.A("boolean", true), s.A("boolean", true)), true)
	rtc(s.A("OR", s.A("boolean", false), s.A("boolean", true)), true)
}

func (s *CollectionTestSuite) ctype() string {
	if s.isArray {
		return "array"
	} else {
		return "object"
	}
}

func (s *CollectionTestSuite) P() rql.ASTNode {
	if s.isArray {
		return Array()
	} else {
		return Object()
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

// EPV => ElementPredicateValue
func (s *CollectionTestSuite) EPV(elem interface{}) interface{} {
	if s.isArray {
		return []interface{}{elem}
	} else {
		return map[string]interface{}{"0": elem}
	}
}
