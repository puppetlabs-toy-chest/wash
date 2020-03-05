package ast

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

// This file contains real-world meta primary test cases

type MetaPrimaryRealWorldTestSuite struct {
	asttest.Suite
	e rql.Entry
	s *rql.EntrySchema
}

// TTC => TrueTestCase
func (s *MetaPrimaryRealWorldTestSuite) TTC(key interface{}, predicate interface{}) {
	q := Query()
	s.MUM(q, s.A("meta", s.A("object", s.A(s.A("key", key), predicate))))
	s.Suite.EETTC(q, s.e)
	// The entry schema predicate should also be true
	s.Suite.EESTTC(q, s.s)
}

func (s *MetaPrimaryRealWorldTestSuite) TestMetaPrimary() {
	s.TTC("architecture", s.A("string", s.A("=", "x86_64")))
	s.TTC("architecture",
		s.A("string",
			s.A("AND",
				s.A("=", "x86_64"),
				s.A("glob", "x86*"),
			),
		),
	)

	// This basically tests that the "amiLaunchIndex" key exists
	s.TTC("amiLaunchIndex", s.A("OR", nil, s.A("NOT", nil)))

	s.TTC("lastModifiedTime",
		s.A("time",
			s.A("AND",
				s.A("=", "2018-10-01T17:37:05Z"),
				s.A("<", "2018-10-02T17:37:05Z"),
			),
		),
	)
	// Should return true because "lastModifiedTime" is not a numeric value.
	// This test is mainly here to test the schema predicate
	s.TTC("lastModifiedTime",
		s.A("NOT",
			s.A("number", s.A("AND", s.A("<=", "0"), s.A(">=", "0"))),
		),
	)
	// This mainly tests the schema predicate, specifically that OR'ing things
	// together works
	s.TTC("lastModifiedTime",
		s.A("OR",
			s.A("boolean", true),
			s.A("time",
				s.A("AND",
					s.A("=", "2018-10-01T17:37:05Z"),
					s.A("<", "2018-10-02T17:37:05Z"),
				),
			),
		),
	)

	s.TTC("blockDeviceMappings",
		s.A("array",
			s.A("some",
				s.A("object",
					s.A(s.A("key", "deviceName"),
						s.A("string", s.A("=", "/dev/sda1"))),
				),
			),
		),
	)
	// Should return true because every element in the blockDeviceMappings
	// array is an object which is NOT an array. This test is mainly here
	// to test the schema predicate
	s.TTC("blockDeviceMappings",
		s.A("array",
			s.A("some",
				s.A("NOT",
					s.A("array", s.A("some", nil)),
				),
			),
		),
	)

	s.TTC("cpuOptions",
		s.A("object",
			s.A(s.A("key", "coreCount"),
				s.A("number", s.A("=", "4")),
			),
		),
	)
	s.TTC("cpuOptions",
		s.A("object",
			s.A(s.A("key", "coreCount"),
				s.A("number",
					s.A("OR",
						s.A("=", "4"),
						s.A("AND",
							s.A("<", "1"),
							s.A(">", "5"),
						),
					),
				),
			),
		),
	)

	s.TTC("tags",
		s.A("array",
			s.A("some",
				s.A("AND",
					s.A("object",
						s.A(s.A("key", "key"),
							s.A("string", s.A("=", "termination_date"))),
					),
					s.A("object",
						s.A(s.A("key", "value"),
							s.A("time", s.A("<", "2017-08-07T13:55:25.680464+00:00"))),
					),
				),
			),
		),
	)

	s.TTC("tags",
		s.A("array",
			s.A("all",
				s.A("object",
					s.A(s.A("key", "key"),
						s.A("string", s.A("NOT", s.A("=", "foo")))),
				),
			),
		),
	)

	s.TTC("networkInterfaces",
		s.A("array",
			s.A("some",
				s.A("AND",
					s.A("object",
						s.A(s.A("key", "association"),
							s.A("object",
								s.A(s.A("key", "ipOwnerID"),
									s.A("string", s.A("=", "amazon"))),
							),
						),
					),
					s.A("object",
						s.A(s.A("key", "privateIpAddresses"),
							s.A("array",
								s.A("some",
									s.A("object",
										s.A(s.A("key", "association"),
											s.A("object",
												s.A(s.A("key", "ipOwnerID"),
													s.A("string", s.A("=", "amazon"))),
											),
										),
									),
								),
							),
						),
					),
				),
			),
		),
	)
}

func TestMetaPrimaryRealWorld(t *testing.T) {
	s := new(MetaPrimaryRealWorldTestSuite)

	rawMeta, err := ioutil.ReadFile("testdata/metadata.json")
	if err != nil {
		t.Fatal(fmt.Sprintf("Failed to read testdata/metadata.json"))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rawMeta, &m); err != nil {
		t.Fatal(fmt.Sprintf("Failed to unmarshal testdata/metadata.json: %v", err))
	}
	s.e.Metadata = m

	rawMetaSchema, err := ioutil.ReadFile("testdata/metadataSchema.json")
	if err != nil {
		t.Fatal(fmt.Sprintf("Failed to read testdata/metadataSchema.json"))
	}
	var metaSchema *plugin.JSONSchema
	if err := json.Unmarshal(rawMetaSchema, &metaSchema); err != nil {
		t.Fatal(fmt.Sprintf("Failed to unmarshal testdata/metadata.json: %v", err))
	}
	s.s = &rql.EntrySchema{}
	s.s.SetMetadataSchema(metaSchema)

	suite.Run(t, s)
}
