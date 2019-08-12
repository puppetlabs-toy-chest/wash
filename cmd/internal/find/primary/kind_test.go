package primary

import (
	"testing"

	"github.com/gobwas/glob"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/stretchr/testify/suite"
)

type KindPrimaryTestSuite struct {
	primaryTestSuite
}

func (s *KindPrimaryTestSuite) TestErrors() {
	s.RETC("", "requires additional arguments")
	s.RETC("[a", "invalid pattern: unexpected end of input")
}

func (s *KindPrimaryTestSuite) TestValidInput() {
	// Test the entry predicate
	s.RNTC("a", "", types.Entry{})
	// Test the main schema predicate
	s.RSTC("containers*container", "", "docker/containers/container", "docker/containers/container/fs")
}

func (s *KindPrimaryTestSuite) TestKindP() {
	g, err := glob.Compile("containers*container")
	if s.NoError(err) {
		p := kindP(g, false)

		// Test the entry predicate
		entry := types.Entry{}
		s.False(p.P(types.Entry{}))
		entry.Schema = &types.EntrySchema{}
		s.True(p.P(entry))

		// Test the schema predicate
		schema := &types.EntrySchema{}
		schema.SetPath("docker/containers/container")
		s.True(p.SchemaP().P(schema))
		schema.SetPath("docker/containers/container/fs")
		s.False(p.SchemaP().P(schema))

		// Ensure that the predicate requires entry schemas
		s.True(p.SchemaRequired())

		// Ensure that the predicate returns false if schema
		// is the stree root
		schema.SetPath("docker")
		s.False(p.SchemaP().P(schema))
	}
}

func (s *KindPrimaryTestSuite) TestKindP_Negate() {
	g, err := glob.Compile("containers*container")
	if s.NoError(err) {
		p := kindP(g, false).Negate().(types.EntryPredicate)

		// Test the entry predicate
		entry := types.Entry{}
		s.False(p.P(types.Entry{}))
		entry.Schema = &types.EntrySchema{}
		s.True(p.P(entry))

		// Test the schema predicate
		schema := &types.EntrySchema{}
		schema.SetPath("docker/containers/container")
		s.False(p.SchemaP().P(schema))
		schema.SetPath("docker/containers/container/fs")
		s.True(p.SchemaP().P(schema))

		// Ensure that the predicate still requires entry schemas
		s.True(p.SchemaRequired())

		// Ensure that the predicate still returns false if schema
		// is the stree root
		schema.SetPath("docker")
		s.False(p.SchemaP().P(schema))
	}
}

func TestKindPrimary(t *testing.T) {
	s := new(KindPrimaryTestSuite)
	s.Parser = Kind
	s.SchemaPParser = types.EntryPredicateParser(Kind.parseFunc).ToSchemaPParser()
	s.ConstructEntry = func(v interface{}) types.Entry {
		return v.(types.Entry)
	}
	s.ConstructEntrySchema = func(v interface{}) *types.EntrySchema {
		s := &types.EntrySchema{}
		s.SetPath(v.(string))
		return s
	}
	suite.Run(t, s)
}
