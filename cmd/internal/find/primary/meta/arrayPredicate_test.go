package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type ArrayPredicateTestSuite struct {
	predicate.ParserTestSuite
}

func (s *ArrayPredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *ArrayPredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateErrors() {
	s.RunTestCases(
		s.NPETC("", `expected an opening '\['`, true),
		s.NPETC("]", `expected an opening '\['`, false),
		s.NPETC("f", `expected an opening '\['`, true),
		s.NPETC("[", `expected a closing '\]'`, false),
		s.NPETC("[a", `expected a closing '\]'`, false),
		s.NPETC("[]", `expected a '\*', '\?', or an array index inside '\[\]'`, false),
		s.NPETC("[*a]", `expected a closing '\]' after '\*'`, false),
		s.NPETC("[?a]", `expected a closing '\]' after '\?'`, false),
		s.NPETC("[a]", `expected an array index inside '\[\]'`, false),
		s.NPETC("[-15]", `expected an array index inside '\[\]'`, false),
		s.NPETC("[?]-true", `expected a '\.' or '\[' after '\]' but got -true instead`, false),
		s.NPETC("[?]", `expected a predicate after \[\?\]`, false),
		s.NPETC("[*]", `expected a predicate after \[\*\]`, false),
		s.NPETC("[15]", `expected a predicate after \[15\]`, false),
		s.NPETC("[?] +{", "expected.*closing.*}", false),
	)
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateValidInput() {
	mp := make(map[string]interface{})
	mp["key"] = true
	s.RunTestCases(
		// Test -empty
		s.NPTC("-empty", "", []interface{}{}),
		// Test each of the possible arrayPs
		s.NPTC("[?] -true -size", "-size", toA(false, true)),
		s.NPTC("[*] -true -size", "-size", toA(true, true)),
		s.NPTC("[0] -true -size", "-size", toA(true)),
		// Test key sequences
		s.NPTC("[?][?] -true -size", "-size", toA(toA(true))),
		s.NPTC("[?].key -true -size", "-size", toA(mp)),
	)
}

func (s *ArrayPredicateTestSuite) TestParseArrayPredicateType() {
	// These test only the valid inputs. The error cases are tested in
	// TestParseArrayPredicateErrors.

	ptype, token, err := parseArrayPredicateType("[?]")
	if s.NoError(err) {
		s.Equal(byte('s'), ptype.t)
		s.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[*]")
	if s.NoError(err) {
		s.Equal(byte('a'), ptype.t)
		s.Equal("", token)
	}

	ptype, token, err = parseArrayPredicateType("[15]")
	if s.NoError(err) {
		s.Equal(byte('n'), ptype.t)
		s.Equal(uint(15), ptype.n)
		s.Equal("", token)
	}
}

func (s *ArrayPredicateTestSuite) TestArrayPSome() {
	p := arrayP(arrayPredicateType{t: 's'}, trueP)

	// Returns false for a non-array value
	s.False(p("foo"))

	// Returns false if no elements satisfy p
	s.False(p(toA(false)))

	// Returns true if some element satifies p
	s.True(p(toA(true)))
	s.True(p(toA("foo", true)))
}

func (s *ArrayPredicateTestSuite) TestArrayPAll() {
	p := arrayP(arrayPredicateType{t: 'a'}, trueP)

	// Returns false for a non-array value
	s.False(p("foo"))

	// Returns false if no elements satisfy p
	s.False(p(toA(false)))

	// Returns false if only some of the elements satisfy p
	s.False(p(toA(false, true)))

	// Returns true if all the elements satisfy P
	s.True(p(toA(true)))
	s.True(p(toA(true, true)))
}

func (s *ArrayPredicateTestSuite) TestArrayPNth() {
	p := arrayP(arrayPredicateType{t: 'n', n: 1}, trueP)

	// Returns false for a non-array value
	s.False(p("foo"))

	// Returns false if n >= len(vs)
	s.False(p(toA(true)))

	// Returns false if vs[n] does not satisfy p
	s.False(p(toA(true, false)))

	// Returns true if vs[n] satisfies p
	s.True(p(toA(true, true)))
}

func toA(vs ...interface{}) []interface{} {
	return vs
}

func TestArrayPredicate(t *testing.T) {
	s := new(ArrayPredicateTestSuite)
	s.Parser = predicate.GenericParser(parseArrayPredicate)
	suite.Run(t, s)
}
