package meta

import (
	"testing"
	"time"

	"github.com/puppetlabs/wash/cmd/internal/find/params"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type ObjectPredicateTestSuite struct {
	parsertest.Suite
}

func (s *ObjectPredicateTestSuite) SetupTest() {
	params.StartTime = time.Now()
}

func (s *ObjectPredicateTestSuite) TeardownTest() {
	params.StartTime = time.Time{}
}

func (s *ObjectPredicateTestSuite) TestKeyRegex() {
	s.Regexp(keyRegex, "k")
	s.Regexp(keyRegex, "key")
	s.Regexp(keyRegex, "key1.key2")
	s.Regexp(keyRegex, "key1[]")
	s.Regexp(keyRegex, "key1]")
	s.Regexp(keyRegex, "key1[")

	s.NotRegexp(keyRegex, "")
	s.NotRegexp(keyRegex, ".")
	s.NotRegexp(keyRegex, "[")
	s.NotRegexp(keyRegex, "]")
	s.NotRegexp(keyRegex, ".key")
	s.NotRegexp(keyRegex, "[key")
	s.NotRegexp(keyRegex, "]key")
}

func (s *ObjectPredicateTestSuite) TestParseObjectPredicateErrors() {
	s.RunTestCases(
		s.NPETC("", "expected a key sequence", true),
		s.NPETC("foo", "key sequences must begin with a '.'", true),
		s.NPETC(".", "expected a key sequence after '.'", false),
		s.NPETC(".[", "expected a key sequence after '.'", false),
		s.NPETC(".key", "expected a predicate after key", false),
		s.NPETC(".key -foo", "expected a predicate after key", false),
		s.NPETC(".key +{", "expected.*closing.*}", false),
		s.NPETC(".key]", `expected an opening '\['`, false),
		s.NPETC(".key[", `expected a closing '\]'`, false),
		// Test predicate expression errors
		s.NPETC(".key )", `\): no beginning '\('`, false),
		s.NPETC(".key (", `\(: missing closing '\)'`, false),
		s.NPETC(".key ( -true", `\(: missing closing '\)'`, false),
		s.NPETC(".key ( )", `\(\): empty inner expression`, false),
		s.NPETC(".key ( -true -false -foo", "unknown predicate -foo", false),
	)
}

func (s *ObjectPredicateTestSuite) TestParseObjectPredicateValidInput() {
	// Make the satisfying maps
	mp1 := make(map[string]interface{})
	mp1["key"] = true

	mp2 := make(map[string]interface{})
	mp2["key1"] = make(map[string]interface{})
	(mp2["key1"].(map[string]interface{}))["key2"] = true

	mp3 := make(map[string]interface{})
	mp3["key"] = toA(true)

	// Run the tests
	s.RunTestCases(
		// Test -empty
		s.NPTC("-empty -size", "-size", make(map[string]interface{})),
		// Test a non-key sequence
		s.NPTC(".key -true -size", "-size", mp1),
		// Test an object key sequence
		s.NPTC(".key1.key2 -true -size", "-size", mp2),
		// Test an array key sequence
		s.NPTC(".key[?] -true -size", "-size", mp3),
		// Now test predicate expressions. The predicate expression parser's
		// already well tested, so these are just some sanity checks.
		s.NPNTC(".key ( -true -a -false ) -size", "-size", mp1),
		s.NPTC(".key ( -true -o -false ) -size", "-size", mp1),
		s.NPTC(".key ( ! -false ) -size", "-size", mp1),
		s.NPTC(".key ( ! ( -true -a -false ) ) -size", "-size", mp1),
	)
}

func (s *ObjectPredicateTestSuite) TestObjectP_NotAnObject() {
	objP := objectP("foo", trueP)
	s.False(objP.IsSatisfiedBy("not an object"))
	s.False(objP.Negate().IsSatisfiedBy("not an object"))
}

func (s *ObjectPredicateTestSuite) TestObjectP_NonexistantKey() {
	mp := make(map[string]interface{})
	objP := objectP("foo", trueP)
	s.False(objP.IsSatisfiedBy(mp))
	s.False(objP.Negate().IsSatisfiedBy(mp))
}

func (s *ObjectPredicateTestSuite) TestObjectP_ExistantKey() {
	mp := make(map[string]interface{})
	mp["foo"] = "baz"

	var calledP bool
	p := genericPredicate(func(v interface{}) bool {
		calledP = true
		s.Equal("baz", v, "objectP did not pass-in mp[key] into p")
		return true
	})
	objP := objectP("foo", p)

	s.True(objP.IsSatisfiedBy(mp), "objectP did not return p(mp[key])")
	s.True(calledP, "objectP did not invoke p")
	
	// Now test negation
	calledP = false
	s.False(objP.Negate().IsSatisfiedBy(mp), "objectP.Negate() did not return !p(mp[key])")
	s.True(calledP, "objectP.Negate() did not invoke p")
}

func (s *ObjectPredicateTestSuite) TestFindMatchingKey() {
	mp := make(map[string]interface{})
	mp["foo"] = "bar"
	mp["baz"] = "baz"

	s.Equal("foo", findMatchingKey(mp, "Foo"))
	s.Equal("foo", findMatchingKey(mp, "foo"))
	s.Equal("", findMatchingKey(mp, "nonexistant_key"))
}

func TestObjectPredicate(t *testing.T) {
	s := new(ObjectPredicateTestSuite)
	s.Parser = predicate.ToParser(parseObjectPredicate)
	suite.Run(t, s)
}
