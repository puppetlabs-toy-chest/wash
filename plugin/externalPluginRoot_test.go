package plugin

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ExternalPluginRootTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginRootTestSuite) TestInit() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntryBase(),
		script:    mockScript,
	}}

	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(
			mock.Anything,
			"init",
			nil,
		).Return(stdout, err).Once()
	}

	// Test that if InvokeAndWait errors, then Init returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	err := root.Init(map[string]interface{}{})
	suite.EqualError(mockErr, err.Error())

	// Test that Init returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	err = root.Init(map[string]interface{}{})
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Init returns an error if the root does not implement
	// "list"
	mockInvokeAndWait([]byte("{\"name\":\"root\",\"methods\":[]}"), nil)
	err = root.Init(map[string]interface{}{})
	suite.Regexp("implement.*list", err)

	// Test that Init properly decodes the root from stdout
	stdout := "{\"name\":\"foo\",\"methods\":[\"list\"]}"
	mockInvokeAndWait([]byte(stdout), nil)
	err = root.Init(map[string]interface{}{})
	if suite.NoError(err) {
		expectedRoot := &externalPluginRoot{
			externalPluginEntry: &externalPluginEntry{
				EntryBase:        NewEntryBase(),
				methods:          []string{"list"},
				script:           root.script,
			},
		}
		expectedRoot.SetName("foo")

		suite.Equal(expectedRoot, root)
	}
}

func TestExternalPluginRoot(t *testing.T) {
	suite.Run(t, new(ExternalPluginRootTestSuite))
}
