package asttest

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/internal/errz"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"

	"github.com/stretchr/testify/suite"
)

// Suite represents a type that tests RQL AST nodes. N represents the AST node
// constructor that returns an empty AST node object. It can be overridden by
// each test. DefaultN represents the default node constructor that should be
// set when the suite class is created.
//
// TODO: Add some more comments once a tested version of this is working.
type Suite struct {
	suite.Suite
	DefaultNodeConstructor func() rql.ASTNode
	NodeConstructor        func() rql.ASTNode
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

func (s *Suite) SetupTest() {
	s.NodeConstructor = s.DefaultNodeConstructor
}

// MTC => MarshalTestCase
func (s *Suite) MTC(n rql.ASTNode, expected interface{}) {
	s.Equal(expected, n.Marshal())
}

// UMETC => UmarshalErrorTestCase
func (s *Suite) UMETC(ast interface{}, errRegex string, isMatchErr bool) {
	ast = s.toUnmarshaledJSON(ast)
	if err := s.NodeConstructor().Unmarshal(ast); s.Error(err) {
		if isMatchErr {
			s.True(errz.IsMatchError(err), "err is not a MatchError")
		} else {
			s.False(errz.IsMatchError(err), "err is a MatchError")
		}
		s.Regexp(errRegex, err)
	}
}

// EVTTC => EvalValueTrueTestCases
func (s *Suite) EVTTC(ast interface{}, trueVs ...interface{}) {
	for _, trueV := range trueVs {
		// Entry metadata values are passed into value predicates. These are unmarshaled from
		// JSON. Thus to ensure that the value predicates do the right type assertions, we also
		// convert each input to their unmarshaled JSON go types. So this means that if we have
		// something like trueV == int64(10), then s.toUnmarshaledJSON(trueV) would return
		// float64(10) since json.Unmarshal converts all numbers to float64 by default.
		s.True(s.constructNode(ast).(rql.ValuePredicate).EvalValue(s.toUnmarshaledJSON(trueV)))
	}
}

// EVFTC => EvalValueFalseTestCases
func (s *Suite) EVFTC(ast interface{}, falseVs ...interface{}) {
	for _, falseV := range falseVs {
		// See the comment in EVTTC to understand why s.toUnmarshaledJSON is necessary.
		s.False(s.constructNode(ast).(rql.ValuePredicate).EvalValue(s.toUnmarshaledJSON(falseV)))
	}
}

// EVSTTC => EvalValueSchemaTrueTestCases
func (s *Suite) EVSTTC(ast interface{}, trueVs ...map[string]interface{}) {
	for _, trueV := range s.ToJSONSchemas(trueVs...) {
		s.True(s.constructNode(ast).(rql.ValuePredicate).EvalValueSchema(trueV))
	}
}

// EVSFTC => EvalValueSchemaFalseTestCases
func (s *Suite) EVSFTC(ast interface{}, falseVs ...map[string]interface{}) {
	for _, falseV := range s.ToJSONSchemas(falseVs...) {
		s.False(s.constructNode(ast).(rql.ValuePredicate).EvalValueSchema(falseV))
	}
}

func (s *Suite) ToJSONSchemas(schemas ...map[string]interface{}) []*plugin.JSONSchema {
	jsonSchemas := []*plugin.JSONSchema{}
	for _, schema := range schemas {
		rawJSON, err := json.Marshal(schema)
		if err != nil {
			s.FailNow(fmt.Sprintf("Error encoding schema %v: %v", schema, err))
		}
		var jsonSchema *plugin.JSONSchema
		if err := json.Unmarshal(rawJSON, &jsonSchema); err != nil {
			s.FailNow(fmt.Sprintf("Error decoding schema %v: %v", schema, err))
		}
		jsonSchemas = append(jsonSchemas, jsonSchema)
	}
	return jsonSchemas
}

// ESTTC => EvalStringTrueTestCases
func (s *Suite) ESTTC(ast interface{}, trueVs ...string) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.StringPredicate).EvalString(trueV))
	}
}

// ESFTC => EvalStringFalseTestCases
func (s *Suite) ESFTC(ast interface{}, falseVs ...string) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.StringPredicate).EvalString(falseV))
	}
}

// ENTTC => EvalNumericTrueTestCases
func (s *Suite) ENTTC(ast interface{}, trueVs ...decimal.Decimal) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.NumericPredicate).EvalNumeric(trueV))
	}
}

// ENFTC => EvalNumericFalseTestCases
func (s *Suite) ENFTC(ast interface{}, falseVs ...decimal.Decimal) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.NumericPredicate).EvalNumeric(falseV))
	}
}

// ETTTC => EvalTimeTrueTestCases
func (s *Suite) ETTTC(ast interface{}, trueVs ...time.Time) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.TimePredicate).EvalTime(trueV))
	}
}

// ETFTC => EvalTimeFalseTestCases
func (s *Suite) ETFTC(ast interface{}, falseVs ...time.Time) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.TimePredicate).EvalTime(falseV))
	}
}

// EETTC => EvalEntryTrueTestCases
func (s *Suite) EETTC(ast interface{}, trueVs ...rql.Entry) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.EntryPredicate).EvalEntry(trueV))
	}
}

// EEFTC => EvalEntryFalseTestCases
func (s *Suite) EEFTC(ast interface{}, falseVs ...rql.Entry) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.EntryPredicate).EvalEntry(falseV))
	}
}

// EESTTC => EvalEntrySchemaTrueTestCases
func (s *Suite) EESTTC(ast interface{}, trueVs ...*rql.EntrySchema) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.EntrySchemaPredicate).EvalEntrySchema(trueV))
	}
}

// EESFTC => EvalEntrySchemaFalseTestCases
func (s *Suite) EESFTC(ast interface{}, falseVs ...*rql.EntrySchema) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.EntrySchemaPredicate).EvalEntrySchema(falseV))
	}
}

// EATTC => EvalActionTrueTestCases
func (s *Suite) EATTC(ast interface{}, trueVs ...plugin.Action) {
	for _, trueV := range trueVs {
		s.True(s.constructNode(ast).(rql.ActionPredicate).EvalAction(trueV))
	}
}

// EAFTC => EvalActionFalseTestCases
func (s *Suite) EAFTC(ast interface{}, falseVs ...plugin.Action) {
	for _, falseV := range falseVs {
		s.False(s.constructNode(ast).(rql.ActionPredicate).EvalAction(falseV))
	}
}

// MUM => MustUnmarshal is a wrapper to ASTNode#Unmarshal. It will fail the
// test if unmarshaling fails
func (s *Suite) MUM(n rql.ASTNode, input interface{}) {
	if err := n.Unmarshal(input); err != nil {
		s.FailNow(fmt.Sprintf("unexpectedly failed to unmarshal n: %v", err.Error()))
	}
}

func (s *Suite) constructNode(ast interface{}) rql.ASTNode {
	if _, ok := ast.(rql.ASTNode); ok {
		s.FailNow(fmt.Sprintf("ast %v cannot be an rql.ASTNode", ast))
	}
	n := s.NodeConstructor()
	if err := n.Unmarshal(s.toUnmarshaledJSON(ast)); err != nil {
		s.FailNow(fmt.Sprintf("unexpectedly failed to unmarshal the ast into P: %v", err))
	}
	return n
}

func (s *Suite) toUnmarshaledJSON(input interface{}) interface{} {
	rawJSON, err := json.Marshal(input)
	if err != nil {
		s.FailNow(fmt.Sprintf("unexpectedly failed to marshal %v to JSON: %v", input, err))
	}
	var unmarshaledInput interface{}
	if err := json.Unmarshal(rawJSON, &unmarshaledInput); err != nil {
		s.FailNow(fmt.Sprintf("unexpectedly failed to unmarshal back the input %v from JSON: %v", input, err))
	}
	return unmarshaledInput
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

func (s *Suite) AssertNotImplemented(ast interface{}, interfaceCs ...InterfaceCode) {
	n := s.constructNode(ast)
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
