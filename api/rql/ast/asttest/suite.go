package asttest

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/suite"
)

// Suite represents a type that tests RQL AST nodes
type Suite struct {
	suite.Suite
}

// A ("Array") is a helper meant to make []interface{} input
// specs more readable. For example, instead of []interface{"foo", "bar", "baz"},
// you can use s.A("foo", "bar", "baz")
func (s *Suite) A(vs ...interface{}) []interface{} {
	return vs
}

// N ("Number") is a helper meant to make decimal.NewFromString
// input specs more readable. For example, instead of
// decimal.NewFromString("3"), you can use s.N("3").
func (s *Suite) N(n string) decimal.Decimal {
	d, err := decimal.NewFromString(n)
	if err != nil {
		panic(fmt.Sprintf("s.N unexpected error: %v", err))
	}
	return d
}

// TM ("Time") is a helper meant to make time.Unix input
// specs more readable. For example, instead of
// time.Unix(1000, 0), you can use s.T(1000).
func (s *Suite) TM(t int64) time.Time {
	return time.Unix(t, 0)
}

// MTC => MarshalTestCase
func (s *Suite) MTC(n rql.ASTNode, expected interface{}) {
	s.Equal(expected, n.Marshal())
}

// UMETC => UmarshalErrorTestCase
func (s *Suite) UMETC(n rql.ASTNode, input interface{}, errRegex string, isMatchErr bool) {
	if err := n.Unmarshal(input); s.Error(err) {
		if isMatchErr {
			s.True(errz.IsMatchError(err), "err is not a MatchError")
		} else {
			s.False(errz.IsMatchError(err), "err is a MatchError")
		}
		s.Regexp(errRegex, err)
	}
}

// UMTC => UmarshalTestCase
func (s *Suite) UMTC(n rql.ASTNode, input interface{}, expected rql.ASTNode) {
	if s.NoError(n.Unmarshal(input)) {
		if nt, ok := n.(internal.NonterminalNode); ok {
			n = nt.MatchedNode()
		}
		s.Equal(expected, n)
	}
}

// EVTTC => EvalValueTrueTestCases
func (s *Suite) EVTTC(n rql.ASTNode, trueVs ...interface{}) {
	for _, trueV := range trueVs {
		s.True(n.(rql.ValuePredicate).EvalValue(trueV))
	}
}

// EVFTC => EvalValueFalseTestCases
func (s *Suite) EVFTC(n rql.ASTNode, falseVs ...interface{}) {
	for _, falseV := range falseVs {
		s.False(n.(rql.ValuePredicate).EvalValue(falseV))
	}
}

// ESTTC => EvalStringTrueTestCases
func (s *Suite) ESTTC(n rql.ASTNode, trueVs ...string) {
	for _, trueV := range trueVs {
		s.True(n.(rql.StringPredicate).EvalString(trueV))
	}
}

// ESFTC => EvalStringFalseTestCases
func (s *Suite) ESFTC(n rql.ASTNode, falseVs ...string) {
	for _, falseV := range falseVs {
		s.False(n.(rql.StringPredicate).EvalString(falseV))
	}
}

// ENTTC => EvalNumericTrueTestCases
func (s *Suite) ENTTC(n rql.ASTNode, trueVs ...decimal.Decimal) {
	for _, trueV := range trueVs {
		s.True(n.(rql.NumericPredicate).EvalNumeric(trueV))
	}
}

// ENFTC => EvalNumericFalseTestCases
func (s *Suite) ENFTC(n rql.ASTNode, falseVs ...decimal.Decimal) {
	for _, falseV := range falseVs {
		s.False(n.(rql.NumericPredicate).EvalNumeric(falseV))
	}
}

// ETTTC => EvalTimeTrueTestCases
func (s *Suite) ETTTC(t rql.ASTNode, trueVs ...time.Time) {
	for _, trueV := range trueVs {
		s.True(t.(rql.TimePredicate).EvalTime(trueV))
	}
}

// ETFTC => EvalTimeFalseTestCases
func (s *Suite) ETFTC(t rql.ASTNode, falseVs ...time.Time) {
	for _, falseV := range falseVs {
		s.False(t.(rql.TimePredicate).EvalTime(falseV))
	}
}

// EETTC => EvalEntryTrueTestCases
func (s *Suite) EETTC(e rql.ASTNode, trueVs ...rql.Entry) {
	for _, trueV := range trueVs {
		s.True(e.(rql.EntryPredicate).EvalEntry(trueV))
	}
}

// EEFTC => EvalEntryFalseTestCases
func (s *Suite) EEFTC(e rql.ASTNode, falseVs ...rql.Entry) {
	for _, falseV := range falseVs {
		s.False(e.(rql.EntryPredicate).EvalEntry(falseV))
	}
}

// EESTTC => EvalEntrySchemaTrueTestCases
func (s *Suite) EESTTC(e rql.ASTNode, trueVs ...*rql.EntrySchema) {
	for _, trueV := range trueVs {
		s.True(e.(rql.EntrySchemaPredicate).EvalEntrySchema(trueV))
	}
}

// EESFTC => EvalEntrySchemaFalseTestCases
func (s *Suite) EESFTC(e rql.ASTNode, falseVs ...*rql.EntrySchema) {
	for _, falseV := range falseVs {
		s.False(e.(rql.EntrySchemaPredicate).EvalEntrySchema(falseV))
	}
}

// EATTC => EvalActionTrueTestCases
func (s *Suite) EATTC(e rql.ASTNode, trueVs ...plugin.Action) {
	for _, trueV := range trueVs {
		s.True(e.(rql.ActionPredicate).EvalAction(trueV))
	}
}

// EAFTC => EvalActionFalseTestCases
func (s *Suite) EAFTC(e rql.ASTNode, falseVs ...plugin.Action) {
	for _, falseV := range falseVs {
		s.False(e.(rql.ActionPredicate).EvalAction(falseV))
	}
}

// MUM => MustUnmarshal is a wrapper to ASTNode#Unmarshal. It will fail the
// test if unmarshaling fails
func (s *Suite) MUM(n rql.ASTNode, input interface{}) {
	if err := n.Unmarshal(input); err != nil {
		s.FailNow(fmt.Sprintf("unexpectedly failed to unmarshal n: %v", err.Error()))
	}
}

type InterfaceCode int8

// Here, C => Code. So EntryPredicateC => EntryPredicateCode
const (
	PrimaryC InterfaceCode = iota
	EntryPredicateC
	EntrySchemaPredicateC
	ValuePredicateC
	StringPredicateC
	NumericPredicateC
	TimePredicateC
	ActionPredicateC
)

func (s *Suite) AssertNotImplemented(n rql.ASTNode, interfaceCs ...InterfaceCode) {
	for _, interfaceC := range interfaceCs {
		switch interfaceC {
		case PrimaryC:
			s.FailNow("AssertNotImplemented should take the *Predicate interfaces' interface codes, _not_ Primary")
		case EntryPredicateC:
			s.Panics(func() { n.(rql.EntryPredicate).EvalEntry(rql.Entry{}) }, "EntryPredicate")
		case EntrySchemaPredicateC:
			s.Panics(func() { n.(rql.EntrySchemaPredicate).EvalEntrySchema(&rql.EntrySchema{}) }, "EntrySchemaPredicate")
		case ValuePredicateC:
			s.Panics(func() { n.(rql.ValuePredicate).EvalValue(nil) }, "ValuePredicate")
		case StringPredicateC:
			s.Panics(func() { n.(rql.StringPredicate).EvalString("") }, "StringPredicate")
		case NumericPredicateC:
			s.Panics(func() { n.(rql.NumericPredicate).EvalNumeric(s.N("0")) }, "NumericPredicate")
		case TimePredicateC:
			s.Panics(func() { n.(rql.TimePredicate).EvalTime(s.TM(0)) }, "TimePredicate")
		case ActionPredicateC:
			s.Panics(func() { n.(rql.ActionPredicate).EvalAction(plugin.Action{}) }, "ActionPredicate")
		}
	}
}
