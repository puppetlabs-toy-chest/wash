package primary

import (
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/api/rql/internal/predicate"
	"github.com/puppetlabs/wash/api/rql/internal/predicate/expression"
	"github.com/stretchr/testify/suite"
)

type MetaTestSuite struct {
	asttest.Suite
}

func (s *MetaTestSuite) TestMarshal() {
	p := Meta(predicate.Object())
	input := s.A("meta", s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true))))
	s.MUM(p, input)
	s.MTC(p, input)
}

func (s *MetaTestSuite) TestUnmarshalErrors() {
	n := Meta(predicate.Object())
	s.UMETC(n, "foo", `meta.*formatted.*"meta".*PE ObjectPredicate`, true)
	s.UMETC(n, s.A("foo", s.A("object", s.A(s.A("key", "foo"), true))), `meta.*formatted.*"meta".*PE ObjectPredicate`, true)
	s.UMETC(n, s.A("meta", "foo", "bar"), `meta.*formatted.*"meta".*PE ObjectPredicate`, false)
	s.UMETC(n, s.A("meta"), `meta.*formatted.*"meta".*PE ObjectPredicate.*missing.*PE ObjectPredicate`, false)
	s.UMETC(n, s.A("meta", s.A("object")), "meta.*PE ObjectPredicate.*element", false)
}

func (s *MetaTestSuite) TestEvalEntry() {
	p := Meta(predicate.Object())
	s.MUM(p, s.A("meta", s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true)))))
	e := rql.Entry{}
	e.Metadata = map[string]interface{}{"foo": false}
	s.EEFTC(p, e)
	e.Metadata["foo"] = true
	s.EETTC(p, e)
}

func (s *MetaTestSuite) TestEvalEntrySchema() {
	p := Meta(predicate.Object())
	s.MUM(p, s.A("meta", s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true)))))
	schema := &rql.EntrySchema{}
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"bar": map[string]interface{}{},
		},
	})[0])
	s.EESFTC(p, schema)
	schema.SetMetadataSchema(nil)
	s.EESTTC(p, schema)
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "boolean",
			},
		},
	})[0])
	s.EESTTC(p, schema)

}

func (s *MetaTestSuite) TestExpression_Atom() {
	expr := expression.New("meta", false, func() rql.ASTNode {
		return Meta(predicate.Object())
	})

	s.MUM(expr, s.A("meta", s.A("object", s.A(s.A("key", "foo"), s.A("boolean", true)))))
	e := rql.Entry{}
	e.Metadata = map[string]interface{}{}
	s.EEFTC(expr, e)
	e.Metadata = map[string]interface{}{"foo": false}
	s.EEFTC(expr, e)
	e.Metadata["foo"] = true
	s.EETTC(expr, e)

	schema := &rql.EntrySchema{}
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"bar": map[string]interface{}{},
		},
	})[0])
	s.EESFTC(expr, schema)
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "boolean",
			},
		},
	})[0])
	s.EESTTC(expr, schema)

	s.AssertNotImplemented(
		expr,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestMeta(t *testing.T) {
	suite.Run(t, new(MetaTestSuite))
}
