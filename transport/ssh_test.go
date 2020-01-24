package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	gssh "github.com/gliderlabs/ssh"
	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type SSHTestSuite struct {
	suite.Suite
	s                        gssh.Server
	m                        mock.Mock
	knownHosts, identityFile string
}

var _ = suite.SetupAllSuite(&SSHTestSuite{})
var _ = suite.SetupTestSuite(&SSHTestSuite{})
var _ = suite.TearDownAllSuite(&SSHTestSuite{})
var _ = suite.TearDownTestSuite(&SSHTestSuite{})

const (
	host = "localhost"
	port = 2222
)

// Must be mocked for successful connections.
func (suite *SSHTestSuite) Handler(s gssh.Session) {
	suite.m.Called(s)
}

// Must be mocked if Identity.Password is set.
func (suite *SSHTestSuite) PasswordHandler(ctx gssh.Context, password string) bool {
	return suite.m.Called(ctx, password).Bool(0)
}

// Must be mocked.
func (suite *SSHTestSuite) PublicKeyHandler(ctx gssh.Context, key gssh.PublicKey) bool {
	return suite.m.Called(ctx, key).Bool(0)
}

func generateSigner() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(key)
}

func createKnownHosts(address string, key ssh.PublicKey) (string, error) {
	tmpfile, err := ioutil.TempFile("", "wash_ssh_knownhosts")
	if err != nil {
		return "", err
	}
	if _, err = tmpfile.Write([]byte(knownhosts.Line([]string{address}, key))); err != nil {
		return "", err
	}
	return tmpfile.Name(), tmpfile.Close()
}

func (suite *SSHTestSuite) SetupSuite() {
	// Generate a host key and knownhosts file
	signer, err := generateSigner()
	if err != nil {
		suite.T().Fatal(err)
	}

	addr := host + ":" + strconv.Itoa(port)
	knownHosts, err := createKnownHosts(addr, signer.PublicKey())
	if err != nil {
		suite.T().Fatal(err)
	}
	suite.knownHosts = knownHosts

	// Generate an identity file
	tmpfile, err := ioutil.TempFile("", "wash_ssh_key")
	if err != nil {
		suite.T().Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		suite.T().Fatal(err)
	}
	privBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(tmpfile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		suite.T().Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		suite.T().Fatal(err)
	}
	suite.identityFile = tmpfile.Name()

	// Setup SSH server
	suite.s.Addr = addr
	suite.s.Handle(suite.Handler)
	suite.s.AddHostKey(signer)
	suite.s.PasswordHandler = suite.PasswordHandler
	suite.s.PublicKeyHandler = suite.PublicKeyHandler
	go func() { suite.T().Log(suite.s.ListenAndServe()) }()
}

func (suite *SSHTestSuite) SetupTest() {
	// Reset mocks before every test
	suite.m = mock.Mock{}
}

func (suite *SSHTestSuite) TearDownTest() {
	connectionCache.Flush()
}

func (suite *SSHTestSuite) TearDownSuite() {
	// Teardown SSH server
	if err := suite.s.Close(); err != nil {
		suite.T().Log(err)
	}
	if err := os.Remove(suite.knownHosts); err != nil {
		suite.T().Log(err)
	}
	if err := os.Remove(suite.identityFile); err != nil {
		suite.T().Log(err)
	}
}

func (suite *SSHTestSuite) Identity() Identity {
	return Identity{Host: host, Port: port, IdentityFile: suite.identityFile, KnownHosts: suite.knownHosts}
}

func (suite *SSHTestSuite) TestExec() {
	var user, command string
	suite.m.On("Handler", mock.Anything).Run(func(args mock.Arguments) {
		sess := args.Get(0).(gssh.Session)
		user = sess.User()
		command = sess.RawCommand()
		_, err := sess.Write([]byte("hello\n"))
		suite.NoError(err)
		suite.NoError(sess.CloseWrite())
	})
	suite.m.On("PublicKeyHandler", mock.Anything, mock.Anything).Return(true)

	cmd, err := ExecSSH(context.Background(), suite.Identity(), []string{"echo", "hello"}, plugin.ExecOptions{})
	if suite.NoError(err) {
		var resp []plugin.ExecOutputChunk
		for chunk := range cmd.OutputCh() {
			resp = append(resp, chunk)
		}
		exit, err := cmd.ExitCode()
		suite.NoError(err)
		suite.Zero(exit)
		suite.Len(resp, 1)

		suite.Equal("root", user)
		suite.Equal("echo hello", command)
	}
}

func (suite *SSHTestSuite) TestExec_WithUser() {
	var user string
	suite.m.On("Handler", mock.Anything).Run(func(args mock.Arguments) {
		user = args.Get(0).(gssh.Session).User()
	})
	suite.m.On("PublicKeyHandler", mock.Anything, mock.Anything).Return(true)

	identity := suite.Identity()
	identity.User = "other"
	cmd, err := ExecSSH(context.Background(), identity, []string{"echo", "hello"}, plugin.ExecOptions{})
	if suite.NoError(err) {
		<-cmd.OutputCh()
		exit, err := cmd.ExitCode()
		suite.NoError(err)
		suite.Zero(exit)
		suite.Equal("other", user)
	}
}

func (suite *SSHTestSuite) TestExec_WithPassword() {
	password := "password"
	suite.m.On("Handler", mock.Anything).Run(func(args mock.Arguments) {})
	suite.m.On("PublicKeyHandler", mock.Anything, mock.Anything).Return(false)
	suite.m.On("PasswordHandler", mock.Anything, password).Return(true)

	identity := suite.Identity()
	identity.Password = password
	cmd, err := ExecSSH(context.Background(), identity, []string{}, plugin.ExecOptions{})
	if suite.NoError(err) {
		<-cmd.OutputCh()
		exit, err := cmd.ExitCode()
		suite.NoError(err)
		suite.Zero(exit)
	}
}

func (suite *SSHTestSuite) TestExec_ContextCancelled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	suite.m.On("Handler", mock.Anything).Run(func(args mock.Arguments) {})
	suite.m.On("PublicKeyHandler", mock.Anything, mock.Anything).Return(true)

	cmd, err := ExecSSH(ctx, suite.Identity(), []string{}, plugin.ExecOptions{})
	if suite.NoError(err) {
		<-cmd.OutputCh()
		_, err := cmd.ExitCode()
		suite.EqualError(err, "failed to fetch the command's exit code: context canceled")
	}
}

func TestClient(t *testing.T) {
	suite.Run(t, new(SSHTestSuite))
}
