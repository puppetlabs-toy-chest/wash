package meta

import (
	"fmt"
	"strings"
)

/*
keySequence represents a metadata key sequence. It is used by schema
predicates to generate metadata schema predicates. keySequence is an
immutable type.

Note that key sequences are built from the bottom-up (i.e. after each
recursive call). For example, the key sequence specified by ".key1.key2 5"
will have the following call sequence:
	ks.
		EndsWithPrimitiveValue().
		AddObject("key2").
		AddObject("key1")
*/
type keySequence struct {
	// If len(segments[i]) <= 0, then segments[i] is an array. Otherwise, segments[i]
	// is an object with key "segments[i]".
	segments       []string
	endValueType   string
	checkExistence bool
}

const (
	object    = "object"
	array     = "array"
	primitive = "primitive"
)

// AddObject adds an object with the specified key to ks
func (ks keySequence) AddObject(key string) keySequence {
	if len(key) <= 0 {
		panic("ks.AddObject called with an empty key")
	}
	// We only care about matching keys, which is the first key
	// s.t. upcase(matching_key) == upcase(key).
	return ks.add(strings.ToUpper(key))
}

// AddArray adds an array to ks
func (ks keySequence) AddArray() keySequence {
	return ks.add("")
}

// EndsWithPrimitiveValue indicates that the ks will end with
// a primitive value
func (ks keySequence) EndsWithPrimitiveValue() keySequence {
	return ks.setEndValueType(primitive)
}

// EndsWithObject indicates that the ks will end with an object.
func (ks keySequence) EndsWithObject() keySequence {
	return ks.setEndValueType(object)
}

// EndsWithArray indicates that the ks will end with an array
func (ks keySequence) EndsWithArray() keySequence {
	return ks.setEndValueType(array)
}

// CheckExistence indicates that the ks will be checked for existence,
// meaning its endValueType is irrelevant
func (ks keySequence) CheckExistence() keySequence {
	ks2 := ks.clone()
	ks2.checkExistence = true
	return ks2
}

/*
toJSON converts ks to its JSON representation. For example, if the key
sequence is something like ".bar[].baz[][]" and it ended with a primitive
value, then the returned JSON would be
	{
		"BAR": [
			{
				"BAZ": [
					[
						null
					]
				]
			}
		]
	}
*/
func (ks keySequence) toJSON() interface{} {
	// Helper to make the code more readable
	isArray := func(segment string) bool {
		return len(segment) <= 0
	}

	if len(ks.endValueType) <= 0 {
		panic("ks.toJSONObject(): called without setting the end value type")
	}

	// Iteratively construct the JSON object by updating the current value
	// after each iteration.
	var obj interface{}
	var currentValue interface{}
	setCurrentValueTo := func(newValue interface{}) {
		if currentValue == nil {
			// This is the initial state. Note that obj will contain our
			// serialized key sequence
			obj = newValue
		} else {
			switch t := currentValue.(type) {
			case []interface{}:
				t[0] = newValue
			case map[string]interface{}:
				// Since we don't know t's key, use a for-loop to access it.
				for k := range t {
					t[k] = newValue
				}
			}
		}
		currentValue = newValue
	}
	for _, segment := range ks.segments {
		var newValue interface{}
		if isArray(segment) {
			newValue = make([]interface{}, 1)
		} else {
			mp := make(map[string]interface{})
			mp[segment] = nil
			newValue = mp
		}
		setCurrentValueTo(newValue)
	}
	// Now ensure that the end value's properly set
	var endValue interface{}
	switch ks.endValueType {
	case object:
		endValue = make(map[string]interface{})
	case array:
		endValue = make([]interface{}, 0)
	default:
		endValue = nil
	}
	setCurrentValueTo(endValue)

	return obj
}

func (ks keySequence) setEndValueType(endValueType string) keySequence {
	if len(ks.endValueType) > 0 {
		msg := fmt.Sprintf("ks.setEndValueType: called with an already-set end value type of %v", ks.endValueType)
		panic(msg)
	}
	ks2 := ks.clone()
	ks2.endValueType = endValueType
	return ks2
}

func (ks keySequence) add(segment string) keySequence {
	ks2 := ks.clone()
	// Key sequences are built from the bottom-up so we need to
	// prepend here.
	ks2.segments = append([]string{segment}, ks2.segments...)
	return ks2
}

func (ks keySequence) clone() keySequence {
	ks2 := keySequence{}
	ks2.segments = make([]string, len(ks.segments))
	copy(ks2.segments, ks.segments)
	ks2.endValueType = ks.endValueType
	ks2.checkExistence = ks.checkExistence
	return ks2
}
