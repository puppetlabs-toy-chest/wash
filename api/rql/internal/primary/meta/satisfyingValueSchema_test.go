package meta

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type SatisfyingValueSchemaTestSuite struct {
	suite.Suite
}

// ISVS => IntermediateSVS

func (s *SatisfyingValueSchemaTestSuite) TestEmptyISVS() {
	s.runTestCase(NewSatisfyingValueSchema(), func(v interface{}) interface{} {
		return v
	})
}

func (s *SatisfyingValueSchemaTestSuite) TestSingleObjectISVS() {
	s.runTestCase(
		(NewSatisfyingValueSchema()).AddObject("foo"),
		func(v interface{}) interface{} {
			return map[string]interface{}{
				"FOO": v,
			}
		},
	)
}

func (s *SatisfyingValueSchemaTestSuite) TestSingleArrayISVS() {
	s.runTestCase(
		(NewSatisfyingValueSchema()).AddArray(),
		func(v interface{}) interface{} {
			return []interface{}{v}
		},
	)
}

func (s *SatisfyingValueSchemaTestSuite) TestNestedISVS() {
	isvs := (NewSatisfyingValueSchema()).
		AddObject("foo").
		AddObject("bar").
		AddArray().
		AddObject("baz").
		AddArray().
		AddArray()

	eRVG := func(v interface{}) interface{} {
		return map[string]interface{}{
			"FOO": map[string]interface{}{
				"BAR": []interface{}{
					map[string]interface{}{
						"BAZ": []interface{}{
							[]interface{}{v},
						},
					},
				},
			},
		}
	}

	s.runTestCase(isvs, eRVG)
}

func TestSatisfyingValueSchema(t *testing.T) {
	suite.Run(t, new(SatisfyingValueSchemaTestSuite))
}

// isvs => intermediateSVS, eRVG => expectedRepresentativeValueGenerator
func (s *SatisfyingValueSchemaTestSuite) runTestCase(isvs SatisfyingValueSchema, eRVG func(interface{}) interface{}) {
	svs := isvs.EndsWithObject()
	s.Equal([]interface{}{eRVG(map[string]interface{}{})}, svs.representativeValues)

	svs = isvs.EndsWithArray()
	s.Equal([]interface{}{eRVG([]interface{}{})}, svs.representativeValues)

	svs = isvs.EndsWithPrimitiveValue()
	s.Equal([]interface{}{eRVG(nil)}, svs.representativeValues)

	svs = isvs.EndsWithAnything()
	s.Equal([]interface{}{eRVG(map[string]interface{}{}), eRVG([]interface{}{}), eRVG(nil)}, svs.representativeValues)
}
