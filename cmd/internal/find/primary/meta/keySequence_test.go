package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// NOTE: Remember that key sequences are built from the bottom-up,
// _not_ top-down.

type KeySequenceTestSuite struct {
	suite.Suite
}

func (s *KeySequenceTestSuite) TestAddObject() {
	ks := keySequence{}
	ks2 := ks.AddObject("foo")
	s.Equal([]string{"FOO"}, ks2.segments)
	// Ensure KeySequence's immutability
	s.Equal(([]string)(nil), ks.segments)
}

func (s *KeySequenceTestSuite) TestAddArray() {
	ks := keySequence{}
	ks2 := ks.AddArray()
	s.Equal([]string{""}, ks2.segments)
	// Ensure KeySequence's immutability
	s.Equal(([]string)(nil), ks.segments)
}

func (s *KeySequenceTestSuite) TestEndsWithPrimitiveValue() {
	ks := keySequence{}
	ks2 := ks.EndsWithPrimitiveValue()
	s.Equal(primitive, ks2.endValueType)
	// Ensure KeySequence's immutability
	s.Equal("", ks.endValueType)
}

func (s *KeySequenceTestSuite) TestEndsWithObject() {
	ks := keySequence{}
	ks2 := ks.EndsWithObject()
	s.Equal(object, ks2.endValueType)
	// Ensure KeySequence's immutability
	s.Equal("", ks.endValueType)
}

func (s *KeySequenceTestSuite) TestEndsWithArray() {
	ks := keySequence{}
	ks2 := ks.EndsWithArray()
	s.Equal(array, ks2.endValueType)
	// Ensure KeySequence's immutability
	s.Equal("", ks.endValueType)
}

func (s *KeySequenceTestSuite) TestCheckExistence() {
	ks := keySequence{}
	ks2 := ks.CheckExistence()
	s.True(ks2.checkExistence)
	// Ensure KeySequence's immutability
	s.False(ks.checkExistence)
}

func (s *KeySequenceTestSuite) TestToJSON_EmptyKeySequence() {
	ks := (keySequence{}).EndsWithObject()
	s.Equal(map[string]interface{}{}, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleObject_EndsWithObject() {
	ks := (keySequence{}).EndsWithObject().AddObject("foo")
	expected := map[string]interface{}{
		"FOO": map[string]interface{}{},
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleObject_EndsWithArray() {
	ks := (keySequence{}).EndsWithArray().AddObject("foo")
	expected := map[string]interface{}{
		"FOO": []interface{}{},
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleObject_EndsWithPrimitiveValue() {
	ks := (keySequence{}).EndsWithPrimitiveValue().AddObject("foo")
	expected := map[string]interface{}{
		"FOO": nil,
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleArray_EndsWithObject() {
	ks := (keySequence{}).EndsWithObject().AddArray()
	expected := []interface{}{
		make(map[string]interface{}),
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleArray_EndsWithArray() {
	ks := (keySequence{}).EndsWithArray().AddArray()
	expected := []interface{}{
		[]interface{}{},
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_SingleArray_EndsWithPrimitiveValue() {
	ks := (keySequence{}).EndsWithPrimitiveValue().AddArray()
	expected := []interface{}{
		nil,
	}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_Nested_EndsWithObject() {
	ks, expected, lastValue := s.nestedKSTestCase((keySequence{}).EndsWithObject())
	lastValue[0] = make(map[string]interface{})
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_Nested_EndsWithArray() {
	ks, expected, lastValue := s.nestedKSTestCase((keySequence{}).EndsWithArray())
	lastValue[0] = []interface{}{}
	s.Equal(expected, ks.toJSON())
}

func (s *KeySequenceTestSuite) TestToJSON_ValidKeySequence_Nested_EndsWithPrimitiveValue() {
	ks, expected, lastValue := s.nestedKSTestCase((keySequence{}).EndsWithPrimitiveValue())
	lastValue[0] = nil
	s.Equal(expected, ks.toJSON())
}

func TestKeySequence(t *testing.T) {
	suite.Run(t, new(KeySequenceTestSuite))
}

// This is for the ToJSON_Nested* tests
func (s *KeySequenceTestSuite) nestedKSTestCase(ks keySequence) (keySequence, map[string]interface{}, []interface{}) {
	ks = ks.
		AddArray().
		AddArray().
		AddObject("baz").
		AddArray().
		AddObject("bar").
		AddObject("foo")

	lastValue := []interface{}{nil}
	expected := map[string]interface{}{
		"FOO": map[string]interface{}{
			"BAR": []interface{}{
				map[string]interface{}{
					"BAZ": []interface{}{
						lastValue,
					},
				},
			},
		},
	}

	return ks, expected, lastValue
}
