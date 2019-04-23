package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ObjectPredicateTestSuite struct {
	ParserTestSuite
}

func (suite *ObjectPredicateTestSuite) TestKeyRegex() {
	suite.Regexp(keyRegex, "k")
	suite.Regexp(keyRegex, "key")

	suite.NotRegexp(keyRegex, "")
	suite.NotRegexp(keyRegex, ".")
	suite.NotRegexp(keyRegex, "[")
	suite.NotRegexp(keyRegex, "]")
}

func (suite *ObjectPredicateTestSuite) TestErrors() {
	suite.runTestCases(
		nPETC("", "expected a key sequence", true),
		nPETC("foo", "key sequences must begin with a '.'", true),
		nPETC(".", "expected a key sequence after '.'", false),
		nPETC(".key", "expected a predicate after key", false),
		nPETC(".key +{", "expected.*closing.*}", false),
	)
}

func (suite *ObjectPredicateTestSuite) TestValidInput() {
	// Make the satisfying maps
	mp1 := make(map[string]interface{})
	mp1["key"] = true

	mp2 := make(map[string]interface{})
	mp2["key1"] = make(map[string]interface{})
	(mp2["key1"].(map[string]interface{}))["key2"] = true

	mp3 := make(map[string]interface{})
	mp3["key"] = toA(true)

	// Run the tests
	suite.runTestCases(
		// Test -empty
		nPTC("-empty -size", "-size", make(map[string]interface{})),
		// Test a non-key sequence
		nPTC(".key -true -size", "-size", mp1),
		// Test an object key sequence
		nPTC(".key1.key2 -true -size", "-size", mp2),
		// Test an array key sequence
		nPTC(".key[] -true -size", "-size", mp3),
	)
}

func (suite *ObjectPredicateTestSuite) TestObjectP_NotAnObject() {
	suite.False(objectP("foo", trueP)("not an object"))
}

func (suite *ObjectPredicateTestSuite) TestObjectP_NonexistantKey() {
	mp := make(map[string]interface{})
	suite.False(objectP("foo", trueP)(mp))
}

func (suite *ObjectPredicateTestSuite) TestObjectP_ExistantKey() {
	mp := make(map[string]interface{})
	mp["foo"] = "baz"

	var calledP bool
	p := func(v interface{}) bool {
		calledP = true
		suite.Equal("baz", v, "objectP did not pass-in mp[key] into p")
		return true
	}

	suite.True(objectP("foo", p)(mp), "objectP did not return p(mp[key])")
	suite.True(calledP, "objectP did not invoke p")
}

func (suite *ObjectPredicateTestSuite) TestFindMatchingKey() {
	mp := make(map[string]interface{})
	mp["foo"] = "bar"
	mp["baz"] = "baz"

	suite.Equal("foo", findMatchingKey(mp, "Foo"))
	suite.Equal("foo", findMatchingKey(mp, "foo"))
	suite.Equal("", findMatchingKey(mp, "nonexistant_key"))
}

func TestObjectPredicate(t *testing.T) {
	s := new(ObjectPredicateTestSuite)
	s.parser = parseObjectPredicate
	suite.Run(t, s)
}
