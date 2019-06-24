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
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(
			mock.Anything,
			"init",
			nil,
			"null",
		).Return(stdout, err).Once()
	}

	// Test that if InvokeAndWait errors, then Init returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	err := root.Init(nil)
	suite.EqualError(mockErr, err.Error())

	// Test that Init returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	err = root.Init(nil)
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Init properly decodes the root from stdout
	stdout := "{}"
	mockInvokeAndWait([]byte(stdout), nil)
	err = root.Init(nil)
	if suite.NoError(err) {
		expectedRoot := &externalPluginRoot{
			externalPluginEntry: &externalPluginEntry{
				EntryBase: NewEntry("foo"),
				methods:   []string{"list"},
				script:    root.script,
			},
		}

		suite.Equal(expectedRoot, root)
	}
}

func (suite *ExternalPluginRootTestSuite) TestInitWithConfig() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &externalPluginRoot{&externalPluginEntry{
		EntryBase: NewEntry("foo"),
		script:    mockScript,
	}}

	mockScript.OnInvokeAndWait(
		mock.Anything,
		"init",
		nil,
		`{"key":["value"]}`,
	).Return([]byte("{}"), nil).Once()

	suite.NoError(root.Init(map[string]interface{}{"key": []string{"value"}}))
}

func TestExternalPluginRoot(t *testing.T) {
	suite.Run(t, new(ExternalPluginRootTestSuite))
}
