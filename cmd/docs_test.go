package cmd

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	apitypes "github.com/puppetlabs/wash/api/types"
	"github.com/stretchr/testify/suite"
)

type DocsTestSuite struct {
	suite.Suite
}

func (suite *DocsTestSuite) TestStringifySupportedAttributes() {
	toRegexStr := func(t time.Time) string {
		return regexp.QuoteMeta(t.String())
	}

	path := "foo"
	entry := apitypes.Entry{}
	entry.Attributes.
		SetAtime(time.Now()).
		SetMtime(time.Now()).
		SetCtime(time.Now()).
		SetCrtime(time.Now()).
		SetSize(10)

	supportedAttributes := stringifySupportedAttributes(path, entry)

	suite.Regexp("^SUPPORTED ATTRIBUTES", supportedAttributes)
	suite.Regexp(fmt.Sprintf("atime.*last access time.*%v", toRegexStr(entry.Attributes.Atime())), supportedAttributes)
	suite.Regexp(fmt.Sprintf("mtime.*last modified time.*%v", toRegexStr(entry.Attributes.Mtime())), supportedAttributes)
	suite.Regexp(fmt.Sprintf("ctime.*change time.*%v", toRegexStr(entry.Attributes.Ctime())), supportedAttributes)
	suite.Regexp(fmt.Sprintf("crtime.*creation time.*%v", toRegexStr(entry.Attributes.Crtime())), supportedAttributes)
	suite.Regexp(fmt.Sprintf("size.*%v", entry.Attributes.Size()), supportedAttributes)
	suite.Regexp("meta foo", supportedAttributes)
	suite.Regexp("meta --attribute foo", supportedAttributes)
	suite.Regexp("find --help", supportedAttributes)
}

func (suite *DocsTestSuite) TestStringifySupportedActions() {
	path := "foo"
	entry := apitypes.Entry{
		Actions: []string{
			"list",
			"read",
			"stream",
			"exec",
			"delete",
			"signal",
		},
	}

	// Test the default stuff
	supportedActions := stringifySupportedActions(path, entry)
	suite.Regexp("^SUPPORTED ACTIONS", supportedActions)
	suite.Regexp("list.*\n.*ls foo", supportedActions)
	suite.Regexp("read.*\n.*cat foo", supportedActions)
	suite.Regexp("stream.*\n.*tail -f foo", supportedActions)
	suite.Regexp(`exec.*\n.*wexec foo <command> <args\.\.\.>.*\n.*wexec foo uname`, supportedActions)
	suite.Regexp("delete.*\n.*delete foo", supportedActions)
	suite.Regexp("signal.*\n.*signal <signal> foo.*\n.*signal start foo", supportedActions)

	// Test non-file-like entry
	entry.Actions = []string{"read", "write"}
	supportedActions = stringifySupportedActions(path, entry)
	suite.Regexp("echo 'foo' >> foo.*\n.*write.*chunk.*foo", supportedActions)

	// Test file-like entry
	entry.Attributes.SetSize(10)
	supportedActions = stringifySupportedActions(path, entry)
	suite.Regexp("echo 'foo' >> foo.*\n.*\n.*vim foo", supportedActions)
}

func (suite *DocsTestSuite) TestStringifySignalSet() {
	setName := "SUPPORTED SIGNAL GROUPS"
	signals := []apitypes.SignalSchema{}

	signal := apitypes.SignalSchema{}
	signal.SetName("foo").SetDescription("foo signal")
	signals = append(signals, signal)
	signal = apitypes.SignalSchema{}
	signal.SetName("bar").SetDescription("bar signal")
	signals = append(signals, signal)

	supportedSignals := stringifySignalSet(setName, signals)

	suite.Regexp("^SUPPORTED SIGNAL GROUPS", supportedSignals)
	suite.Regexp(".*foo.*\n.*foo signal", supportedSignals)
	suite.Regexp(".*bar.*\n.*bar signal", supportedSignals)
}

func TestDocs(t *testing.T) {
	suite.Run(t, new(DocsTestSuite))
}
