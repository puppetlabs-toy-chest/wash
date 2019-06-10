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

func (m *mockRoot) Init(cfg map[string]interface{}) error {
	args := m.Called(cfg)
	return args.Error(0)
}

func (m *mockRoot) List(ctx context.Context) ([]Entry, error) {
	args := m.Called(ctx)
	return args.Get(0).([]Entry), args.Error(1)
}

func (m *mockRoot) ChildSchemas() []EntrySchema {
	return []EntrySchema{
		EntrySchema{
			Type: "entry",
		},
	}
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
	m := &mockRoot{EntryBase: NewEntryBase()}
	m.SetName("mine")
	cfg := map[string]interface{}{}
	m.On("Init", cfg).Return(nil)

	suite.NoError(reg.RegisterPlugin(m, cfg))
	m.AssertExpectations(suite.T())
	suite.Contains(reg.Plugins(), "mine")
}

func (suite *RegistryTestSuite) TestRegisterPluginWithConfig() {
	reg := NewRegistry()
	m := &mockRoot{EntryBase: NewEntryBase()}
	m.SetName("mine")
	cfg := map[string]interface{}{"key": "value"}
	m.On("Init", cfg).Return(nil)

	suite.NoError(reg.RegisterPlugin(m, cfg))
	m.AssertExpectations(suite.T())
	suite.Contains(reg.Plugins(), "mine")
}

func (suite *RegistryTestSuite) TestRegisterPluginInitError() {
	reg := NewRegistry()
	m := &mockRoot{EntryBase: NewEntryBase()}
	m.SetName("mine")
	m.On("Init", map[string]interface{}(nil)).Return(errors.New("failed"))

	suite.EqualError(reg.RegisterPlugin(m, nil), "failed")
	m.AssertExpectations(suite.T())
	suite.NotContains(reg.Plugins(), "mine")
}

func (suite *RegistryTestSuite) TestRegisterPluginInvalidPluginName() {
	panicFunc := func() {
		reg := NewRegistry()
		m := &mockRoot{EntryBase: NewEntryBase()}
		m.SetName("b@dname")
		_ = reg.RegisterPlugin(m, map[string]interface{}{})
	}

	suite.Panics(
		panicFunc,
		"r.RegisterPlugin: invalid plugin name b@dname. The plugin name must consist of alphanumeric characters, or a hyphen",
	)
}

func (suite *RegistryTestSuite) TestRegisterPluginRegisteredPlugin() {
	panicFunc := func() {
		reg := NewRegistry()
		m1 := &mockRoot{EntryBase: NewEntryBase()}
		m1.SetName("mine")
		_ = reg.RegisterPlugin(m1, map[string]interface{}{})
		_ = reg.RegisterPlugin(m1, map[string]interface{}{})
	}

	suite.Panics(panicFunc, "r.RegisterPlugin: the mine plugin's already been registered")
}

func (suite *RegistryTestSuite) TestRegisterExternalPlugin() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/external.sh"}

	suite.NoError(reg.RegisterExternalPlugin(spec, map[string]interface{}{}))
	suite.Contains(reg.Plugins(), "external")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginWithConfig() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/external.sh"}

	suite.NoError(reg.RegisterExternalPlugin(spec, map[string]interface{}{"key": "value"}))
	suite.Contains(reg.Plugins(), "external")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNoExec() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/noexec"}

	suite.EqualError(reg.RegisterExternalPlugin(spec, map[string]interface{}{}), "script testdata/noexec is not executable")
	suite.NotContains(reg.Plugins(), "external")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNoExist() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/noexist"}

	suite.EqualError(reg.RegisterExternalPlugin(spec, map[string]interface{}{}), "stat testdata/noexist: no such file or directory")
	suite.NotContains(reg.Plugins(), "external")
}

func (suite *RegistryTestSuite) TestRegisterExternalPluginNotFile() {
	reg := NewRegistry()
	spec := ExternalPluginSpec{Script: "testdata/notfile"}

	suite.EqualError(reg.RegisterExternalPlugin(spec, map[string]interface{}{}), "script testdata/notfile is not a file")
	suite.NotContains(reg.Plugins(), "external")
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
