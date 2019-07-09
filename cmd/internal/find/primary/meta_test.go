package primary

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/suite"
)

type MetaPrimaryTestSuite struct {
	primaryTestSuite
	e types.Entry
	s *types.EntrySchema
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryErrors() {
	s.RETC("", "expected a key sequence")
	s.RETC("foo", "key sequences must begin with a '.'")
	s.RETC(".", "expected a key sequence after '.'")
	s.RETC(".[", "expected a key sequence after '.'")
	s.RETC(".key", "expected a predicate expression")
	s.RETC(".key +{", "expected.*closing.*}")
	s.RETC(".key]", `expected an opening '\['`)
	s.RETC(".key[", `expected a closing '\]'`)
	// Test some inner predicate expression parser errors
	s.RETC(".key1 .key2 (", `\(: missing closing '\)'`)
	s.RETC(".key1 .key2 ( -foo", "unknown predicate -foo")
	s.RETC(".key1 [?] (", `\(: missing closing '\)'`)
	s.RETC(".key1 [?] ( -foo", "unknown predicate -foo")
}

/*
We always want the schema predicate to return true when the predicate's
true. That's why the tests with "RTC" also include a corresponding schemaP
case ("RSTC"). More complex schemaP cases are tested separately.

NOTE: This does not necessarily hold true for false predicates. For example,
if m['key'] == 5, then ".key 6" will return false, but the schema predicate
will still return true since m['key'] == primitive_value.
*/

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputTruePredicates() {
	s.RTC(".architecture x86_64 -primary", "-primary", s.e)
	s.RSTC(".architecture x86_64 -primary", "-primary", s.s)

	s.RTC(".blockDeviceMappings[?] .deviceName /dev/sda1 -primary", "-primary", s.e)
	s.RSTC(".blockDeviceMappings[?] .deviceName /dev/sda1 -primary", "-primary", s.s)

	s.RTC(".cpuOptions.coreCount 4 -primary", "-primary", s.e)
	s.RSTC(".cpuOptions.coreCount 4 -primary", "-primary", s.s)

	s.RTC(".tags[?] .key termination_date -a .value +1h -primary", "-primary", s.e)
	s.RSTC(".tags[?] .key termination_date -a .value +1h -primary", "-primary", s.s)

	s.RTC(".tags[?] .key foo -o .key department -primary", "-primary", s.e)
	s.RSTC(".tags[?] .key foo -o .key department -primary", "-primary", s.s)

	// TODO: Create the corresponding SchemaP test case once
	// https://github.com/puppetlabs/wash/issues/360 is resolved.
	s.RTC(".elasticGpuAssociations -null -primary", "-primary", s.e)

	s.RTC(".networkInterfaces[?] .association.ipOwnerID amazon -a .privateIpAddresses[?] .association.ipOwnerID amazon -primary", "-primary", s.e)
	s.RSTC(".networkInterfaces[?] .association.ipOwnerID amazon -a .privateIpAddresses[?] .association.ipOwnerID amazon -primary", "-primary", s.s)

	// Test some inner predicate expressions

	s.RTC(".tags[?] .key ( foo -o department ) -primary", "-primary", s.e)
	s.RSTC(".tags[?] .key ( foo -o department ) -primary", "-primary", s.s)

	s.RTC(".blockDeviceMappings[?] .ebs ( .attachTime +1h -a .status attached ) -primary", "-primary", s.e)
	s.RSTC(".blockDeviceMappings[?] .ebs ( .attachTime +1h -a .status attached ) -primary", "-primary", s.s)

	s.RTC(".cpuOptions ( .coreCount ( ( -1 -a +5 ) -o 4 ) ) .threadsPerCore 1 -primary", "-primary", s.e)
	s.RSTC(".cpuOptions ( .coreCount ( ( -1 -a +5 ) -o 4 ) ) .threadsPerCore 1 -primary", "-primary", s.s)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputTrueSchemaPredicates() {
	// Should pass b/c objects in networkInterfaces do have an association key
	s.RSTC(".networkInterfaces[?] .association -exists -primary", "-primary", s.s)
	// Should pass b/c objects in securityGroups do have a groupID key
	s.RSTC(".securityGroups[?] ( .groupID 4 -o foo 10 ) -primary", "-primary", s.s)
	// Should be true
	noMetaSchema := &types.EntrySchema{}
	s.RSTC("-empty -primary", "-primary", noMetaSchema)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputFalsePredicates() {
	// Should fail b/c arch key does not exist
	s.RNTC(".arch x86_64 -primary", "-primary", s.e)
	// Should fail b/c architecture is a string, not a number
	s.RNTC(".architecture +10 -primary", "-primary", s.e)
	// Should fail b/c the termination_date tag's value is in the past while
	// +{1h} queries the future.
	s.RNTC(".tags[?] .key termination_date -a .value +{1h} -primary", "-primary", s.e)
	// Should fail b/c the tags array has elements whose ".key" value is _not_ termination_date.
	// Informally, this means that this EC2 instance has more than one tag.
	s.RNTC(".tags[*] .key termination_date -primary", "-primary", s.e)
	// Should fail b/c architecture cannot be both a number and a string
	s.RNTC(".architecture +10 -a x86_64 -primary", "-primary", s.e)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputFalseSchemaPredicates() {
	// Should fail b/c the arch key does not exist
	s.RNSTC(".arch x86_64 -primary", "-primary", s.s)
	// Should fail b/c "platform" is a primitive value, not an empty object/array
	s.RNSTC(".platform -empty -primary", "-primary", s.s)
	// Should fail b/c "placement" cannot be an empty object
	s.RNSTC(".placement -empty -primary", "-primary", s.s)
	// Should fail b/c CPU options has no "NumThreads" key
	s.RNSTC(".cpuOptions.numThreads 4 -primary", "-primary", s.s)
	// Should fail b/c objects in securityGroups do _not_ have both a "groupID"
	// and a "foo" key
	s.RNSTC(".securityGroups[?] ( .groupID 4 -a foo 10 ) -primary", "-primary", s.s)
	// Should fail b/c hibernation options can never be empty
	s.RNSTC(".hiberationOptions -empty -primary", "-primary", s.s)
}

func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputNegation() {
	s.RNTC(".architecture ! x86_64 -primary", "-primary", s.e)
	s.RNTC(".blockDeviceMappings[?] ! .deviceName /dev/sda1 -primary", "-primary", s.e)
	s.RNTC(".cpuOptions.coreCount ! 4 -primary", "-primary", s.e)

	// De'Morgan's Law: !(p1(a) && p2(a)) == ! p1(a) || ! p2(a). Since there's more than one
	// tag (e.g. "department"), the negation of this predicate will evaluate to true.
	s.RTC(".tags[?] ! ( .key termination_date -a .value +1h ) -primary", "-primary", s.e)
	s.RSTC(".tags[?] ! ( .key termination_date -a .value +1h ) -primary", "-primary", s.s)

	// De'Morgan's Law: !(p1(a) || p2(a)) == !p1(a) && !p2(a). Substituting, this translates to
	// "at least one tag that does _not_ have the "key" key set to 'foo' AND 'department'". Since
	// we have a tag with "key" set to "termination_date", and since "termination_date" is not "foo"
	// and "department", this predicate evaluates to true.
	s.RTC(".tags[?] ! ( .key foo -o .key department ) -primary", "-primary", s.e)
	s.RSTC(".tags[?] ! ( .key foo -o .key department ) -primary", "-primary", s.s)

	s.RNTC(".elasticGpuAssociations ! -null -primary", "-primary", s.e)
	// There's only one network interface, so the negation here evaluates to false.
	s.RNTC(".networkInterfaces[?] ! ( .association.ipOwnerID amazon -a .privateIpAddresses[?] .association.ipOwnerID amazon ) -primary", "-primary", s.e)
	// Test negation for inner predicate expressions
	s.RNTC(".tags[0] .key ( ! ( foo -o department ) ) -primary", "-primary", s.e)
	s.RNTC(".cpuOptions ( .coreCount ( ( -1 -a +5 ) -o ! 4 ) ) .threadsPerCore 1 -primary", "-primary", s.e)
}
func (s *MetaPrimaryTestSuite) TestMetaPrimaryValidInputSchemaPredicateNegation() {
	// The schema predicate still requires m['architecture'] == primitive_value, so
	// it should return true. The same sort of reasoning applies for the next few
	// cases as well.
	s.RSTC(".architecture ! x86_64 -primary", "-primary", s.s)
	s.RSTC(".blockDeviceMappings[?] ! .deviceName /dev/sda1 -primary", "-primary", s.s)
	s.RSTC(".tags ! -empty -primary", "-primary", s.s)
	// Using De'Morgan's law, this parses to ".key primitive_value -a .foo primitive_value".
	// Thus, the schema predicate should return false because a tag does not have a "foo"
	// key.
	s.RNSTC(".tags[?] ! ( .key termination_date -o .foo bar ) -primary", "-primary", s.s)
	// This parses to ".key primitive_value -o .foo primitive_value". Hence, the schema
	// predicate should return true because a tag does have a "key" key.
	s.RSTC(".tags[?] ! ( .key termination_date -a .foo bar ) -primary", "-primary", s.s)
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
	s.e.Metadata = m

	rawMetaSchema, err := ioutil.ReadFile("testdata/metadataSchema.json")
	if err != nil {
		t.Fatal(fmt.Sprintf("Failed to read testdata/metadataSchema.json"))
	}
	var metaSchema *plugin.JSONSchema
	if err := json.Unmarshal(rawMetaSchema, &metaSchema); err != nil {
		t.Fatal(fmt.Sprintf("Failed to unmarshal testdata/metadata.json: %v", err))
	}
	s.s = &types.EntrySchema{}
	s.s.SetMetadataSchema(metaSchema)

	s.Parser = Meta
	s.ConstructEntry = func(v interface{}) types.Entry {
		return v.(types.Entry)
	}
	s.SchemaPParser = types.EntryPredicateParser(Meta.parseFunc).ToSchemaPParser()
	s.ConstructEntrySchema = func(v interface{}) *types.EntrySchema {
		return v.(*types.EntrySchema)
	}
	suite.Run(t, s)
}
