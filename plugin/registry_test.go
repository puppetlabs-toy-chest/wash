package plugin

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
}

func (suite *RegistryTestSuite) TestEmptyRegistry() {
	reg := NewRegistry()
	suite.Empty(reg.Plugins())
}

type mockRoot struct {
	EntryBase
	mock.Mock
}

func (m *mockRoot) Init() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockRoot) List(ctx context.Context) ([]Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (suite *RegistryTestSuite) TestPluginNameRegex() {
	suite.Regexp(pluginNameRegex, "a")
	suite.Regexp(pluginNameRegex, "A")
	suite.Regexp(pluginNameRegex, "1")
	suite.Regexp(pluginNameRegex, "_")
	suite.Regexp(pluginNameRegex, "-")
	suite.Regexp(pluginNameRegex, "foobar-123_baz")

	suite.NotRegexp(pluginNameRegex, "")
	suite.NotRegexp(pluginNameRegex, " plugin")
	suite.NotRegexp(pluginNameRegex, "plugin/name")
	suite.NotRegexp(pluginNameRegex, "plugin  ")
}

func (suite *RegistryTestSuite) TestRegisterPlugin() {
	reg := NewRegistry()
	m := &mockRoot{EntryBase: NewRootEntry("mine")}
	m.On("Init").Return(nil)

	suite.NoError(reg.RegisterPlugin(m))
	m.AssertExpectations(suite.T())
	suite.Contains(reg.Plugins(), "mine")
}

func (suite *RegistryTestSuite) TestRegisterPluginInitError() {
	reg := NewRegistry()
	m := &mockRoot{EntryBase: NewRootEntry("mine")}
	m.On("Init").Return(errors.New("failed"))

	suite.EqualError(reg.RegisterPlugin(m), "failed")
	m.AssertExpectations(suite.T())
	suite.NotContains(reg.Plugins(), "mine")
}

func (suite *RegistryTestSuite) TestRegisterPluginInvalidPluginName() {
	panicFunc := func() {
		reg := NewRegistry()
		m := &mockRoot{EntryBase: NewRootEntry("b@dname")}
		_ = reg.RegisterPlugin(m)
	}

	suite.Panics(
		panicFunc,
		"r.RegisterPlugin: invalid plugin name b@dname. The plugin name must consist of alphanumeric characters, or a hyphen",
	)
}

func (suite *RegistryTestSuite) TestRegisterPluginRegisteredPlugin() {
	panicFunc := func() {
		reg := NewRegistry()
		m1 := &mockRoot{EntryBase: NewRootEntry("mine")}
		_ = reg.RegisterPlugin(m1)
		_ = reg.RegisterPlugin(m1)
	}

	suite.Panics(panicFunc, "r.RegisterPlugin: the mine plugin's already been registered")
}

func (suite *RegistryTestSuite) TestRegisterExternalPlugin() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/external.sh"}

	suite.NoError(reg.RegisterExternalPlugin(spec))
	suite.Contains(reg.Plugins(), "test")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNoExec() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/noexec"}

	suite.EqualError(reg.RegisterExternalPlugin(spec), "script testdata/noexec is not executable")
	suite.NotContains(reg.Plugins(), "test")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNoExist() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/noexist"}

	suite.EqualError(reg.RegisterExternalPlugin(spec), "stat testdata/noexist: no such file or directory")
	suite.NotContains(reg.Plugins(), "test")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNotFile() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/notfile"}

	suite.EqualError(reg.RegisterExternalPlugin(spec), "script testdata/notfile is not a file")
	suite.NotContains(reg.Plugins(), "test")
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
