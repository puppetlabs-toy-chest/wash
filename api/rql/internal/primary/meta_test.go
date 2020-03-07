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
	input := s.A("meta", s.A("object", s.A(s.A("key", "foo"), true)))
	s.MUM(p, input)
	s.MTC(p, input)
}

func (s *MetaTestSuite) TestUnmarshalErrors() {
	s.UMETC("foo", `meta.*formatted.*"meta".*PE ObjectPredicate`, true)
	s.UMETC(s.A("foo", s.A("object", s.A(s.A("key", "foo"), true))), `meta.*formatted.*"meta".*PE ObjectPredicate`, true)
	s.UMETC(s.A("meta", "foo", "bar"), `meta.*formatted.*"meta".*PE ObjectPredicate`, false)
	s.UMETC(s.A("meta"), `meta.*formatted.*"meta".*PE ObjectPredicate.*missing.*PE ObjectPredicate`, false)
	s.UMETC(s.A("meta", s.A("object")), "meta.*PE ObjectPredicate.*element", false)
}

func (s *MetaTestSuite) TestEvalEntry() {
	ast := s.A("meta", s.A("object", s.A(s.A("key", "foo"), true)))
	e := rql.Entry{}
	e.Metadata = map[string]interface{}{"foo": false}
	s.EEFTC(ast, e)
	e.Metadata["foo"] = true
	s.EETTC(ast, e)
}

func (s *MetaTestSuite) TestEvalEntrySchema() {
	ast := s.A("meta", s.A("object", s.A(s.A("key", "foo"), true)))
	schema := &rql.EntrySchema{}

	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"bar": map[string]interface{}{},
		},
	})[0])
	s.EESFTC(ast, schema)

	schema.SetMetadataSchema(nil)
	s.EESTTC(ast, schema)

	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "boolean",
			},
		},
	})[0])
	s.EESTTC(ast, schema)

}

func (s *MetaTestSuite) TestExpression_Atom() {
	s.NodeConstructor = func() rql.ASTNode {
		return expression.New("meta", false, func() rql.ASTNode {
			return Meta(predicate.Object())
		})
	}

	ast := s.A("meta", s.A("object", s.A(s.A("key", "foo"), true)))
	e := rql.Entry{}
	e.Metadata = map[string]interface{}{}
	s.EEFTC(ast, e)
	e.Metadata = map[string]interface{}{"foo": false}
	s.EEFTC(ast, e)
	e.Metadata["foo"] = true
	s.EETTC(ast, e)

	schema := &rql.EntrySchema{}
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"bar": map[string]interface{}{},
		},
	})[0])
	s.EESFTC(ast, schema)
	schema.SetMetadataSchema(s.ToJSONSchemas(map[string]interface{}{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{
				"type": "boolean",
			},
		},
	})[0])
	s.EESTTC(ast, schema)

	s.AssertNotImplemented(
		ast,
		asttest.ValuePredicateC,
		asttest.StringPredicateC,
		asttest.NumericPredicateC,
		asttest.TimePredicateC,
		asttest.ActionPredicateC,
	)
}

func TestMeta(t *testing.T) {
	s := new(MetaTestSuite)
	s.DefaultNodeConstructor = func() rql.ASTNode {
		return Meta(predicate.Object())
	}
	suite.Run(t, s)
}
