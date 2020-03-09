package expression

import (
	"fmt"
	"time"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/plugin"
	"github.com/shopspring/decimal"
)

// TCRunFunc => TestCaseRunFunction
type TCRunFunc = func(*BinOpTestSuite, interface{}, interface{})

type BinOpTestSuite struct {
	asttest.Suite
	opName         string
	testEvalMethod func(s *BinOpTestSuite, RFTC TCRunFunc, RTTC TCRunFunc, constructV func(string) interface{})
}

func (s *BinOpTestSuite) TestMarshal() {
	p := s.NodeConstructor()
	s.MTC(p, s.A(s.opName, "", ""))
}

func (s *BinOpTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", fmt.Sprintf(`formatted.*"%v".*<pe>.*<pe>`, s.opName), true)
	s.UMETC(s.A(s.opName, "10", "11", "12"), fmt.Sprintf(`"%v".*<pe>.*<pe>`, s.opName), false)
	s.UMETC(s.A(s.opName), fmt.Sprintf("%v.*LHS.*RHS.*expression", s.opName), false)
	s.UMETC(s.A(s.opName, "10"), fmt.Sprintf("%v.*LHS.*RHS.*expression", s.opName), false)
	s.UMETC(s.A(s.opName, "syntax", "10"), fmt.Sprintf("%v.*LHS.*syntax", s.opName), false)
	s.UMETC(s.A(s.opName, "10", "syntax"), fmt.Sprintf("%v.*RHS.*syntax", s.opName), false)
}

func (s *BinOpTestSuite) TestEvalEntry() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.EEFTC(ast, falseV.(rql.Entry))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.EETTC(ast, trueV.(rql.Entry))
		},
		func(s string) interface{} {
			e := rql.Entry{}
			e.Name = s
			return e
		},
	)
}

func (s *BinOpTestSuite) TestEvalEntrySchema() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.EESFTC(ast, falseV.(*rql.EntrySchema))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.EESTTC(ast, trueV.(*rql.EntrySchema))
		},
		func(s string) interface{} {
			es := &rql.EntrySchema{}
			es.SetPath(s)
			return es
		},
	)
}

func (s *BinOpTestSuite) TestEvalValue() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.EVFTC(ast, falseV)
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.EVTTC(ast, trueV)
		},
		func(s string) interface{} {
			return s
		},
	)
}

func (s *BinOpTestSuite) TestEvalValueSchema() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.EVSFTC(ast, falseV.(map[string]interface{}))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.EVSTTC(ast, trueV.(map[string]interface{}))
		},
		func(s string) interface{} {
			return map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					s: map[string]interface{}{
						"type": "number",
					},
				},
			}
		},
	)
}

func (s *BinOpTestSuite) TestEvalString() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.ESFTC(ast, falseV.(string))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.ESTTC(ast, trueV.(string))
		},
		func(s string) interface{} {
			return s
		},
	)
}

func (s *BinOpTestSuite) TestEvalNumeric() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.ENFTC(ast, falseV.(decimal.Decimal))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.ENTTC(ast, trueV.(decimal.Decimal))
		},
		func(s string) interface{} {
			d, err := decimal.NewFromString(s)
			if err != nil {
				panic(fmt.Sprintf("unexpected error: %v", err))
			}
			return d
		},
	)
}

func (s *BinOpTestSuite) TestEvalTime() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.ETFTC(ast, falseV.(time.Time))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.ETTTC(ast, trueV.(time.Time))
		},
		func(s string) interface{} {
			d, err := decimal.NewFromString(s)
			if err != nil {
				panic(fmt.Sprintf("unexpected error: %v", err))
			}
			return time.Unix(d.IntPart(), 0)
		},
	)
}

func (s *BinOpTestSuite) TestEvalAction() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, ast interface{}, falseV interface{}) {
			s.EAFTC(ast, falseV.(plugin.Action))
		},
		func(s *BinOpTestSuite, ast interface{}, trueV interface{}) {
			s.EATTC(ast, trueV.(plugin.Action))
		},
		func(s string) interface{} {
			return plugin.Action{Name: s}
		},
	)
}
