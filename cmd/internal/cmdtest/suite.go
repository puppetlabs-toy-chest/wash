package cmdtest

import (
	"bytes"
	"io"

	"github.com/puppetlabs/wash/api/client"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/stretchr/testify/suite"
)

// Suite represents a type that tests Wash subcommands
type Suite struct {
	suite.Suite
	Client           *MockClient
	stdout           *bytes.Buffer
	stderr           *bytes.Buffer
	oldStdout        io.Writer
	oldStderr        io.Writer
	oldColoredStderr io.Writer
	oldNewClient     func() client.Client
}

// SetupTest mocks Stdout/Stderr/ColoredStderr/NewClient
func (s *Suite) SetupTest() {
	// Mock Stdout/Stderr
	s.stdout, s.stderr = &bytes.Buffer{}, &bytes.Buffer{}
	s.oldStdout, s.oldStderr, s.oldColoredStderr = cmdutil.Stdout, cmdutil.Stderr, cmdutil.ColoredStderr
	cmdutil.Stdout, cmdutil.Stderr, cmdutil.ColoredStderr = s.stdout, s.stderr, s.stderr
	// Mock the client
	s.Client = &MockClient{}
	s.oldNewClient = cmdutil.NewClient
	cmdutil.NewClient = func() client.Client {
		return s.Client
	}
}

// TearDownTest resets Stdout/Stdout/ColoredStderr/NewClient
func (s *Suite) TearDownTest() {
	// Reset Stdout/Stderr
	s.stdout, s.stderr = nil, nil
	cmdutil.Stdout, cmdutil.Stderr, cmdutil.ColoredStderr = s.oldStdout, s.oldStderr, s.oldColoredStderr
	s.oldStdout, s.oldStderr, s.oldColoredStderr = nil, nil, nil
	// Reset the client
	s.Client = nil
	cmdutil.NewClient = s.oldNewClient
	s.oldNewClient = nil
}

// Stdout returns stdout's content
func (s *Suite) Stdout() string {
	return s.stdout.String()
}

// Stderr returns stderr's content
func (s *Suite) Stderr() string {
	return s.stderr.String()
}
