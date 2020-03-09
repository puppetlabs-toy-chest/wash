package meta

import (
	"fmt"
	"strings"
)

/*
SatisfyingValueSchema represents a high-level schema of a given leaf's
satisfying values. It is used to generate value schema predicates by
validating its representative values against the provided schema using
a JSON schema validator.

SatisfyingValueSchema is an immutable type.
*/
type SatisfyingValueSchema struct {
	generateRepresentativeValue func(interface{}) interface{}
	representativeValues        []interface{}
}

func NewSatisfyingValueSchema() SatisfyingValueSchema {
	return SatisfyingValueSchema{
		generateRepresentativeValue: func(v interface{}) interface{} {
			return v
		},
	}
}

// AddObject adds an object with the specified key to svs
func (svs SatisfyingValueSchema) AddObject(key string) SatisfyingValueSchema {
	if len(key) <= 0 {
		panic("svs.AddObject called with an empty key")
	}
	return svs.add(func(value interface{}) interface{} {
		return map[string]interface{}{
			// We only care about matching keys, which is the first key
			// s.t. upcase(matching_key) == upcase(key).
			strings.ToUpper(key): value,
		}
	})
}

// AddArray adds an array to svs
func (svs SatisfyingValueSchema) AddArray() SatisfyingValueSchema {
	return svs.add(func(value interface{}) interface{} {
		return []interface{}{value}
	})
}

// EndsWithPrimitiveValue indicates that the svs ends with
// a primitive value
func (svs SatisfyingValueSchema) EndsWithPrimitiveValue() SatisfyingValueSchema {
	return svs.endsWith(nil)
}

// EndsWithObject indicates that the svs ends with an
// object
func (svs SatisfyingValueSchema) EndsWithObject() SatisfyingValueSchema {
	return svs.endsWith(map[string]interface{}{})
}

// EndsWithArray indicates that the svs ends with an
// array
func (svs SatisfyingValueSchema) EndsWithArray() SatisfyingValueSchema {
	return svs.endsWith([]interface{}{})
}

// EndsWithAnything indicates that the svs can end with any
// value
func (svs SatisfyingValueSchema) EndsWithAnything() SatisfyingValueSchema {
	return svs.endsWith(
		map[string]interface{}{},
		[]interface{}{},
		nil,
	)
}

func (svs SatisfyingValueSchema) add(segmentRepresentativeValue func(interface{}) interface{}) SatisfyingValueSchema {
	if svs.isComplete() {
		panic(fmt.Sprintf("svs#add: attempting to add to a completed SatisfyingValueSchema %T", svs))
	}
	return SatisfyingValueSchema{
		generateRepresentativeValue: func(endValue interface{}) interface{} {
			return svs.generateRepresentativeValue(segmentRepresentativeValue(endValue))
		},
	}
}

func (svs SatisfyingValueSchema) endsWith(endValues ...interface{}) SatisfyingValueSchema {
	representativeValues := []interface{}{}
	for _, endValue := range endValues {
		representativeValues = append(representativeValues, svs.generateRepresentativeValue(endValue))
	}
	return SatisfyingValueSchema{
		representativeValues: representativeValues,
	}
}

func (svs SatisfyingValueSchema) isComplete() bool {
	return svs.representativeValues != nil
}
