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
type TCRunFunc = func(*BinOpTestSuite, rql.ASTNode, interface{})

type BinOpTestSuite struct {
	asttest.Suite
	opName         string
	newOp          func(rql.ASTNode, rql.ASTNode) rql.ASTNode
	testEvalMethod func(s *BinOpTestSuite, RFTC TCRunFunc, RTTC TCRunFunc, constructV func(string) interface{})
}

func (s *BinOpTestSuite) TestMarshal() {
	p := s.newOp(newMockP("10"), newMockP("11"))
	s.MTC(p, s.A(s.opName, "10", "11"))
}

func (s *BinOpTestSuite) TestUnmarshal() {
	p := s.newOp(&mockPtype{}, &mockPtype{})
	s.UMETC(p, "foo", fmt.Sprintf(`formatted.*"%v".*<pe>.*<pe>`, s.opName), true)
	s.UMETC(p, s.A(s.opName, "10", "11", "12"), fmt.Sprintf(`"%v".*<pe>.*<pe>`, s.opName), false)
	s.UMETC(p, s.A(s.opName), fmt.Sprintf("%v.*LHS.*RHS.*expression", s.opName), false)
	s.UMETC(p, s.A(s.opName, "10"), fmt.Sprintf("%v.*LHS.*RHS.*expression", s.opName), false)
	s.UMETC(p, s.A(s.opName, "syntax", "10"), fmt.Sprintf("%v.*LHS.*syntax", s.opName), false)
	s.UMETC(p, s.A(s.opName, "10", "syntax"), fmt.Sprintf("%v.*RHS.*syntax", s.opName), false)
	s.UMTC(p, s.A(s.opName, "10", "11"), s.newOp(newMockP("10"), newMockP("11")))
}

func (s *BinOpTestSuite) TestEvalEntry() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.EEFTC(n, falseV.(rql.Entry))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.EETTC(n, trueV.(rql.Entry))
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
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.EESFTC(n, falseV.(*rql.EntrySchema))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.EESTTC(n, trueV.(*rql.EntrySchema))
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
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.EVFTC(n, falseV)
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.EVTTC(n, trueV)
		},
		func(s string) interface{} {
			return s
		},
	)
}

func (s *BinOpTestSuite) TestEvalString() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.ESFTC(n, falseV.(string))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.ESTTC(n, trueV.(string))
		},
		func(s string) interface{} {
			return s
		},
	)
}

func (s *BinOpTestSuite) TestEvalNumeric() {
	s.testEvalMethod(
		s,
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.ENFTC(n, falseV.(decimal.Decimal))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.ENTTC(n, trueV.(decimal.Decimal))
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
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.ETFTC(n, falseV.(time.Time))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.ETTTC(n, trueV.(time.Time))
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
		func(s *BinOpTestSuite, n rql.ASTNode, falseV interface{}) {
			s.EAFTC(n, falseV.(plugin.Action))
		},
		func(s *BinOpTestSuite, n rql.ASTNode, trueV interface{}) {
			s.EATTC(n, trueV.(plugin.Action))
		},
		func(s string) interface{} {
			return plugin.Action{Name: s}
		},
	)
}
