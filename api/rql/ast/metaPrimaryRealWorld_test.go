package ast

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/puppetlabs/wash/api/rql"
	"github.com/puppetlabs/wash/api/rql/ast/asttest"
	"github.com/stretchr/testify/suite"
)

// This file contains real-world meta primary test cases
//
// TODO: Once EvalEntrySchema's implemented, add schema predicate
// tests

type MetaPrimaryRealWorldTestSuite struct {
	asttest.Suite
	e rql.Entry
}

// TTC => TrueTestCase
func (s *MetaPrimaryRealWorldTestSuite) TTC(key interface{}, predicate interface{}) {
	q := Query()
	s.MUM(q, s.A("meta", s.A("object", s.A(s.A("key", key), predicate))))
	s.Suite.EETTC(q, s.e)
}

// FTC => FalseTestCase
func (s *MetaPrimaryRealWorldTestSuite) FTC(key interface{}, predicate interface{}) {
	q := Query()
	s.MUM(q, s.A("meta", s.A("object", s.A(s.A("key", key), predicate))))
	s.Suite.EEFTC(q, s.e)
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

	s.TTC("amiLaunchIndex", s.A("OR", nil, s.A("NOT", nil)))

	s.TTC("lastModifiedTime",
		s.A("time",
			s.A("AND",
				s.A("=", "2018-10-01T17:37:05Z"),
				s.A("<", "2018-10-02T17:37:05Z"),
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

	suite.Run(t, s)
}
