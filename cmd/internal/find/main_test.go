package find

import (
	"testing"
	"strings"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/puppetlabs/wash/api/client"
	"github.com/puppetlabs/wash/cmd/internal/cmdtest"
	"github.com/puppetlabs/wash/cmd/internal/find/types"
	"github.com/puppetlabs/wash/cmd/internal/find/parser"
	"github.com/puppetlabs/wash/cmd/internal/find/primary"
	"github.com/puppetlabs/wash/cmd/internal/find/params"
)

type MainTestSuite struct {
	*cmdtest.Suite
	oldNewWalker   func(r parser.Result, conn client.Client) walker
	walker         *mockWalker
}

func (s *MainTestSuite) SetupTest() {
	s.Suite.SetupTest()
	s.oldNewWalker = newWalker
	s.walker = &mockWalker{}
	newWalker = func(r parser.Result, conn client.Client) walker {
		s.walker.walkerImpl = s.oldNewWalker(r, conn).(*walkerImpl)
		return s.walker
	}
}

func (s *MainTestSuite) TearDownTest() {
	s.Suite.TearDownTest()
	newWalker = s.oldNewWalker
	s.walker = nil
	s.oldNewWalker = nil
}

func (s *MainTestSuite) TestMain_HelpRequested() {
	Main([]string{"-help"})
	s.Equal(Usage(), s.Stdout())
}

func (s *MainTestSuite) TestMain_ParseError() {
	s.Equal(1, Main([]string{"-unknown"}))
	s.Regexp("find:.*", s.Stderr())
}

func (s *MainTestSuite) TestMain_ReferenceTime_NoDaystart() {
	s.walker.On("Walk", mock.Anything).Return(true)
	Main([]string{})
	s.NotEqual(0, params.ReferenceTime.Nanosecond())
}

func (s *MainTestSuite) TestMain_ReferenceTime_WithDaystart() {
	s.walker.On("Walk", mock.Anything).Return(true)
	Main([]string{"-daystart"})
	s.Equal(0, params.ReferenceTime.Nanosecond())
}

func (s *MainTestSuite)  TestMain_MetaPrimarySet_MaxdepthSet() {
	s.walker.On("Walk", mock.Anything).Return(true)
	Main([]string{"-maxdepth", "10", "-meta", "-empty"})
	s.Equal(s.walker.opts.Maxdepth, 10)
}

func (s *MainTestSuite)  TestMain_MetaPrimarySet_MaxdepthNotSet() {
	s.walker.On("Walk", mock.Anything).Return(true)
	Main([]string{"-meta", "-empty"})
	s.Equal(s.walker.opts.Maxdepth, 1)
}

func (s *MainTestSuite) TestMain_SinglePath_SuccessfulWalk() {
	s.walker.On("Walk", ".").Return(true)
	s.Equal(0, Main([]string{}))
}

func (s *MainTestSuite) TestMain_SinglePath_UnsuccessfulWalk() {
	s.walker.On("Walk", ".").Return(false)
	s.Equal(1, Main([]string{}))
}

func (s *MainTestSuite) TestMain_MultiplePaths_SuccessfulWalk() {
	s.walker.On("Walk", "foo").Return(true)
	s.walker.On("Walk", "bar").Return(true)
	s.Equal(0, Main([]string{"foo", "bar"}))
	s.walker.AssertCalled(s.T(), "Walk", "foo")
	s.walker.AssertCalled(s.T(), "Walk", "bar")
}

func (s *MainTestSuite) TestMain_MultiplePaths_UnsuccessfulWalk() {
	s.walker.On("Walk", "foo").Return(true)
	s.walker.On("Walk", "bar").Return(false)
	s.Equal(1, Main([]string{"foo", "bar"}))
	s.walker.AssertCalled(s.T(), "Walk", "foo")
	s.walker.AssertCalled(s.T(), "Walk", "bar")
}

func (s *MainTestSuite) TestPrintHelp_NoValue() {
	helpOpt := types.HelpOption{
		HasValue: false,
	}
	printHelp(helpOpt)
	s.Equal(Usage(), s.Stdout())
}

func (s *MainTestSuite) TestPrintHelp_Syntax() {
	helpOpt := types.HelpOption{
		HasValue: true,
		Syntax: true,
	}
	printHelp(helpOpt)
	expectedStdout := strings.Trim(parser.ExpressionSyntaxDescription, "\n") + "\n"
	s.Equal(expectedStdout, s.Stdout())
}

func (s *MainTestSuite) TestPrintHelp_Primary_NoDetailedDescription() {
	helpOpt := types.HelpOption{
		HasValue: true,
		Primary: "true",
	}
	printHelp(helpOpt)
	expectedStdout := primary.True.Usage() + "\n" + primary.True.Description + "\n"
	s.Equal(expectedStdout, s.Stdout())
}

func (s *MainTestSuite) TestPrintHelp_Primary_WithDetailedDescription() {
	helpOpt := types.HelpOption{
		HasValue: true,
		Primary: "meta",
	}
	printHelp(helpOpt)
	expectedStdout := strings.Trim(primary.Meta.DetailedDescription, "\n") + "\n"
	s.Equal(expectedStdout, s.Stdout())
}

func TestMain(t *testing.T) {
	s := new(MainTestSuite)
	s.Suite = new(cmdtest.Suite)
	suite.Run(t, s)
}

type mockWalker struct {
	mock.Mock
	*walkerImpl
}

func (w *mockWalker) Walk(path string) bool {
	args := w.Called(path)
	return args.Get(0).(bool)
}