package primary

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/cmd/internal/find/parser/parsertest"
	"github.com/stretchr/testify/suite"
)

type MetaPrimaryTestSuite struct {
	parsertest.Suite
	e types.Entry
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryErrors() {
	s.RunTestCases(
		s.NPETC("-m", "expected a key sequence", false),
		s.NPETC("-m foo", "key sequences must begin with a '.'", false),
		s.NPETC("-m .", "expected a key sequence after '.'", false),
		s.NPETC("-m .[", "expected a key sequence after '.'", false),
		s.NPETC("-m .key", "expected a predicate expression", false),
		s.NPETC("-m .key +{", "expected.*closing.*}", false),
		s.NPETC("-m .key]", `expected an opening '\['`, false),
		s.NPETC("-m .key[", `expected a closing '\]'`, false),
		// Test some inner predicate expression parser errors
		s.NPETC("-m .key1 .key2 (", `\(: missing closing '\)'`, false),
		s.NPETC("-m .key1 .key2 ( -foo", "unknown predicate -foo", false),
		s.NPETC("-m .key1 [?] (", `\(: missing closing '\)'`, false),
		s.NPETC("-m .key1 [?] ( -foo", "unknown predicate -foo", false),
	)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputTruePredicates() {
	s.RunTestCases(
		s.NPTC("-m .architecture x86_64 -primary", "-primary", s.e),
		s.NPTC("-m .blockDeviceMappings[?] .deviceName /dev/sda1 -primary", "-primary", s.e),
		s.NPTC("-m .cpuOptions.coreCount 4 -primary", "-primary", s.e),
		s.NPTC("-m .tags[?] .key termination_date -a .value +1h -primary", "-primary", s.e),
		s.NPTC("-m .tags[?] .key foo -o .key department -primary", "-primary", s.e),
		s.NPTC("-m .elasticGpuAssociations -null -primary", "-primary", s.e),
		s.NPTC("-m .networkInterfaces[?] .association.ipOwnerID amazon -a .privateIpAddresses[?] .association.ipOwnerID amazon -primary", "-primary", s.e),
		// Test some inner predicate expressions
		s.NPTC("-m .tags[?] .key ( foo -o department ) -primary", "-primary", s.e),
		s.NPTC("-m .blockDeviceMappings[?] .ebs ( .attachTime +1h -a .status attached ) -primary", "-primary", s.e),
		s.NPTC("-m .cpuOptions ( .coreCount ( ( -1 -a +5 ) -o 4 ) ) .threadsPerCore 1 -primary", "-primary", s.e),
	)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputFalsePredicates() {
	s.RunTestCases(
		// Should fail b/c arch key does not exist
		s.NPNTC("-m .arch x86_64 -primary", "-primary", s.e),
		// Should fail b/c architecture is a string, not a number
		s.NPNTC("-m .architecture +10 -primary", "-primary", s.e),
		// Should fail b/c the termination_date tag's value is in the past while
		// +{1h} queries the future. 
		s.NPNTC("-m .tags[?] .key termination_date -a .value +{1h} -primary", "-primary", s.e),
		// Should fail b/c the tags array has elements whose ".key" value is _not_ termination_date.
		// Informally, this means that this EC2 instance has more than one tag.
		s.NPNTC("-m .tags[*] .key termination_date -primary", "-primary", s.e),
		// Should fail b/c architecture cannot be both a number and a string
		s.NPNTC("-m .architecture +10 -a x86_64 -primary", "-primary", s.e),
	)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputNegation() {
	s.RunTestCases(
		s.NPNTC("-m .architecture ! x86_64 -primary", "-primary", s.e),
		s.NPNTC("-m .blockDeviceMappings[?] ! .deviceName /dev/sda1 -primary", "-primary", s.e),
		s.NPNTC("-m .cpuOptions.coreCount ! 4 -primary", "-primary", s.e),
		// De'Morgan's Law: !(p1(a) && p2(a)) == ! p1(a) || ! p2(a). Since there's more than one
		// tag (e.g. "department"), the negation of this predicate will evaluate to true.
		s.NPTC("-m .tags[?] ! ( .key termination_date -a .value +1h ) -primary", "-primary", s.e),
		// De'Morgan's Law: !(p1(a) || p2(a)) == !p1(a) && !p2(a). Substituting, this translates to
		// "at least one tag that does _not_ have the "key" key set to 'foo' AND 'department'". Since
		// we have a tag with "key" set to "termination_date", and since "termination_date" is not "foo"
		// and "department", this predicate evaluates to true.
		s.NPTC("-m .tags[?] ! ( .key foo -o .key department ) -primary", "-primary", s.e),
		s.NPNTC("-m .elasticGpuAssociations ! -null -primary", "-primary", s.e),
		// There's only one network interface, so the negation here evaluates to false.
		s.NPNTC("-m .networkInterfaces[?] ! ( .association.ipOwnerID amazon -a .privateIpAddresses[?] .association.ipOwnerID amazon ) -primary", "-primary", s.e),
		// Test negation for inner predicate expressions
		s.NPNTC("-m .tags[0] .key ( ! ( foo -o department ) ) -primary", "-primary", s.e),
		s.NPNTC("-m .cpuOptions ( .coreCount ( ( -1 -a +5 ) -o ! 4 ) ) .threadsPerCore 1 -primary", "-primary", s.e),
	)
}

func TestMetaPrimary(t *testing.T) {
	s := new(MetaPrimaryTestSuite)
	rawMeta, err := ioutil.ReadFile("testdata/metadata.json")
	if err != nil {
		t.Fatal(fmt.Sprintf("Failed to read testdata/metadata.json"))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(rawMeta, &m); err != nil {
		t.Fatal(fmt.Sprintf("Failed to unmarshal testdata/metadata.json: %v", err))
	}
	s.e.Attributes.SetMeta(m)
	s.Parser = metaPrimary
	suite.Run(t, s)
}
