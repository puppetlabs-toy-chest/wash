package meta

import (
	"regexp"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/parser/predicate"
	"github.com/stretchr/testify/suite"
)

type ObjectPredicateTestSuite struct {
	parserTestSuite
}

func (s *ObjectPredicateTestSuite) TestParseKey() {
	type testCase struct {
		input    string
		key      string
		rem      string
		errRegex string
	}
	testCases := []testCase{
		// Error cases
		{"", "", "", "key sequences must begin with a '.'"},
		{"f", "", "", "key sequences must begin with a '.'"},
		{"[", "", "", "key sequences must begin with a '.'"},
		{"]", "", "", "key sequences must begin with a '.'"},
		{".", "", "", "expected a key sequence after '.'"},
		{"\\.", "", "", "key sequences must begin with a '.'"},
		{"\\[", "", "", "key sequences must begin with a '.'"},
		{"\\]", "", "", "key sequences must begin with a '.'"},
		// Happy cases
		{".k", "k", "", ""},
		{".key", "key", "", ""},
		{".key1.key2", "key1", ".key2", ""},
		{".key1[]", "key1", "[]", ""},
		{".key1]", "key1", "]", ""},
		{".key1[", "key1", "[", ""},
		{".k\\ey", "k\\ey", "", ""},
		{".\\.", ".", "", ""},
		{".\\[", "[", "", ""},
		{".\\]", "]", "", ""},
		{".foo\\.bar\\[baz\\].", "foo.bar[baz]", ".", ""},
		{".foo\\.bar\\[baz\\][", "foo.bar[baz]", "[", ""},
		{".foo\\.bar\\[baz\\]]", "foo.bar[baz]", "]", ""},
		{".k\\\\ey", "k\\\\ey", "", ""},
		{".k\\\\.ey", "k\\.ey", "", ""},
		{".k\\\\\\.ey", "k\\\\.ey", "", ""},
	}

	for _, testCase := range testCases {
		key, rem, err := parseKey(testCase.input)
		if testCase.errRegex != "" {
			errRegex := regexp.MustCompile(testCase.errRegex)
			s.Regexp(errRegex, err)
		} else if s.NoError(err) {
			s.Equal(testCase.key, key)
			s.Equal(testCase.rem, rem)
		}
	}
}

func (s *ObjectPredicateTestSuite) TestParseObjectPredicateErrors() {
	s.RETC("", "expected a key sequence", true)
	s.RETC("foo", "key sequences must begin with a '.'", true)
	s.RETC(".", "expected a key sequence after '.'", false)
	s.RETC(".[", "expected a key sequence after '.'", false)
	s.RETC(".key", "expected a predicate after key", false)
	s.RETC(".key -foo", "expected a predicate after key", false)
	s.RETC(".key +{", "expected.*closing.*}", false)
	s.RETC(".key]", `expected an opening '\['`, false)
	s.RETC(".key[", `expected a closing '\]'`, false)
	// Test predicate expression errors
	s.RETC(".key )", `\): no beginning '\('`, false)
	s.RETC(".key (", `\(: missing closing '\)'`, false)
	s.RETC(".key ( -true", `\(: missing closing '\)'`, false)
	s.RETC(".key ( )", `\(\): empty inner expression`, false)
	s.RETC(".key ( -true -false -foo", "unknown predicate -foo", false)
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

	// Test -empty
	s.RTC("-empty -size", "-size", make(map[string]interface{}))
	// Test a non-key sequence
	s.RTC(".key -true -size", "-size", mp1)
	// Test an object key sequence
	s.RTC(".key1.key2 -true -size", "-size", mp2)
	// Test an array key sequence
	s.RTC(".key[?] -true -size", "-size", mp3)
	// Now test predicate expressions. The predicate expression parser's
	// already well tested, so these are just some sanity checks.
	s.RNTC(".key ( -true -a -false ) -size", "-size", mp1)
	s.RTC(".key ( -true -o -false ) -size", "-size", mp1)
	s.RTC(".key ( ! -false ) -size", "-size", mp1)
	s.RTC(".key ( ! ( -true -a -false ) ) -size", "-size", mp1)
}

func (s *ObjectPredicateTestSuite) TestParseObjectPredicateValidInput_SchemaP() {
	// Test -empty
	s.RSTC("-empty -size", "-size", "o")
	// Test -exists
	s.RSTC(".key -exists -size", "-size", ".key p")
	s.RSTC(".key -exists -size", "-size", ".key o")
	s.RSTC(".key -exists -size", "-size", ".key a")
	// Test a non-key sequence
	s.RSTC(".key -true -size", "-size", ".key p")
	// Test an object key sequence
	s.RSTC(".key1.key2 -true -size", "-size", ".key1.key2 p")
	s.RNSTC(".key1.key2 -true -size", "-size", "p")
	s.RNSTC(".key1.key2 -true -size", "-size", ".key1.key2 o")
	s.RNSTC(".key1.key2 -true -size", "-size", ".key1 p")
	s.RNSTC(".key1.key2 -true -size", "-size", ".key2 p")
	// Test an array key sequence
	s.RSTC(".key[?] -true -size", "-size", ".key[] p")
	s.RNSTC(".key[?] -true -size", "-size", "p")
	s.RNSTC(".key[?] -true -size", "-size", ".key[] o")
	s.RNSTC(".key[?] -true -size", "-size", ".key p")
	s.RNSTC(".key[?] -true -size", "-size", "[] p")
	// Test a key sequence with the empty predicate
	s.RSTC(".key1.key2 -empty -size", "-size", ".key1.key2 o")

	// Now test predicate expressions. In particular, we want to test
	// that nested schema predicates are properly handled

	// This expects m['key1']['key2'] == primitive_value AND m['key1'] == primitive_value,
	// which is impossible.
	s.RNSTC(".key1 ( .key2 -true -a -false ) -size", "-size", ".key1.key2 p")
	s.RNSTC(".key1 ( .key2 -true -a -false ) -size", "-size", ".key1 p")

	// This expects m['key1']['key2'] == primitive_value OR m['key1'] == primitive_value,
	// which is possible.
	s.RSTC(".key1 ( .key2 -true -o -false ) -size", "-size", ".key1.key2 p")
	s.RSTC(".key1 ( .key2 -true -o -false ) -size", "-size", ".key1 p")
	s.RNSTC(".key1 ( .key2 -true -o -false ) -size", "-size", "p")
}

func (s *ObjectPredicateTestSuite) TestObjectP_NotAnObject() {
	objP := objectP("foo", trueP())
	negatedObjP := objP.Negate().(Predicate)

	s.False(objP.IsSatisfiedBy("not an object"))
	s.False(negatedObjP.IsSatisfiedBy("not an object"))
}

func (s *ObjectPredicateTestSuite) TestObjectP_NonexistantKey() {
	mp := make(map[string]interface{})
	objP := objectP("foo", trueP())
	negatedObjP := objP.Negate().(Predicate)
	s.False(objP.IsSatisfiedBy(mp))
	s.False(negatedObjP.IsSatisfiedBy(mp))

	// The schemaPs should still expect the ".foo" key because that
	// is what the user's querying
	s.True(objP.schemaP().IsSatisfiedBy(s.newSchema(".foo p")))
	s.True(negatedObjP.schemaP().IsSatisfiedBy(s.newSchema(".foo p")))
}

func (s *ObjectPredicateTestSuite) TestObjectP_ExistantKey() {
	mp := make(map[string]interface{})
	mp["foo"] = "baz"

	var calledP bool
	p := genericP(func(v interface{}) bool {
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
	s.SetParser(predicate.ToParser(parseObjectPredicate))
	suite.Run(t, s)
}
