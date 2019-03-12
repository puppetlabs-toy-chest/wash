package plugin

import (
	"fmt"
	"regexp"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ExternalPluginRootTestSuite struct {
	suite.Suite
}

func (suite *ExternalPluginRootTestSuite) TestInit() {
	mockScript := &mockExternalPluginScript{path: "plugin_script"}
	root := &ExternalPluginRoot{&ExternalPluginEntry{
		script:   mockScript,
		washPath: "/foo",
	}}

	mockInvokeAndWait := func(stdout []byte, err error) {
		mockScript.OnInvokeAndWait(mock.AnythingOfType("context.Context"), "init").Return(stdout, err).Once()
	}

	// Test that if InvokeAndWait errors, then Init returns its error
	mockErr := fmt.Errorf("execution error")
	mockInvokeAndWait([]byte{}, mockErr)
	err := root.Init()
	suite.EqualError(mockErr, err.Error())

	// Test that Init returns an error if stdout does not have the right
	// output format
	mockInvokeAndWait([]byte("bad format"), nil)
	err = root.Init()
	suite.Regexp(regexp.MustCompile("stdout"), err)

	// Test that Init properly decodes the root from stdout
	stdout := "{\"name\":\"foo\",\"supported_actions\":[\"list\"]}"
	mockInvokeAndWait([]byte(stdout), nil)
	err = root.Init()
	if suite.NoError(err) {
		expectedRoot := &ExternalPluginRoot{
			ExternalPluginEntry: &ExternalPluginEntry{
				name:             "foo",
				supportedActions: []string{"list"},
				cacheConfig:      newCacheConfig(),
				washPath:         "/foo",
				script:           root.script,
			},
		}

		suite.Equal(expectedRoot, root)
	}

}
