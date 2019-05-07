package transport

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/kballard/go-shellquote"
	"github.com/kevinburke/ssh_config"
	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/datastore"
	"github.com/puppetlabs/wash/plugin"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Cache SSH connections for better performance. Re-using SSH connections can significantly speed
// up repeated SSH operations.
var connectionCache = datastore.NewMemCacheWithEvicted(closeConnection)
var expires = 15 * time.Second

func closeConnection(id string, obj interface{}) {
	if client, ok := obj.(*ssh.Client); ok {
		client.Close()
	}
}

func homeDir() (string, error) {
	curUser, err := user.Current()
	if err != nil {
		return "", err
	}

	if curUser.HomeDir == "" {
		return "", fmt.Errorf("the current user %v does not have a home directory", curUser.Name)
	}
	return curUser.HomeDir, nil
}

func newAgent() (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))

	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
}

func getHostKeyCallback() (ssh.HostKeyCallback, error) {
	homedir, err := homeDir()
	if err != nil {
		return nil, err
	}

	return knownhosts.New(filepath.Join(homedir, ".ssh", "known_hosts"))
}

func sshConnect(host, port, user string, strictHostKeyChecking bool) (*ssh.Client, error) {
	connID := user + "@" + host + ":" + port
	// This is a single-use cache, so pass in an empty category.
	obj, err := connectionCache.GetOrUpdate("", connID, expires, true, func() (interface{}, error) {
		agent, err := newAgent()
		if err != nil {
			return nil, fmt.Errorf("Failed to find config from SSH_AUTH_SOCK: %v", err)
		}

		hostKeyCallback := ssh.InsecureIgnoreHostKey()
		if strictHostKeyChecking {
			hostKeyCallback, err = getHostKeyCallback()
			if err != nil {
				return nil, fmt.Errorf("Loading SSH known hosts file: %v", err)
			}
		}

		sshConfig := &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{agent},
			HostKeyCallback: hostKeyCallback,
		}

		return ssh.Dial("tcp", host+":"+port, sshConfig)
	})

	if err != nil {
		return nil, err
	}
	return obj.(*ssh.Client), nil
}

// Identity identifies how to connect to a target.
type Identity struct {
	Host, User string
}

// ExecSSH executes against a target via SSH. It will look up port, user, and other configuration
// by exact hostname match from default SSH config files. Identity can be used to override the
// default user.
//
// Lots of SSH configuration is currently omitted, such as global known hosts files, finding known
// hosts from the config, identity file from config... pretty much everything but port and user
// from config as enumerated in https://github.com/kevinburke/ssh_config/blob/0.5/validators.go.
//
// The known hosts file will be ignored if StrictHostKeyChecking=no, such as in
// ```
// Host *.compute.amazonaws.com
//   StrictHostKeyChecking no
// ```
func ExecSSH(ctx context.Context, id Identity, cmd []string, opts plugin.ExecOptions) (plugin.ExecResult, error) {
	var result plugin.ExecResult

	// find port, username, etc from .ssh/config
	port, err := ssh_config.GetStrict(id.Host, "Port")
	if err != nil {
		return result, err
	}

	user := id.User
	if user == "" {
		if user, err = ssh_config.GetStrict(id.Host, "User"); err != nil {
			return result, err
		}
	}

	if user == "" {
		user = "root"
	}

	strictHostKeyChecking, err := ssh_config.GetStrict(id.Host, "StrictHostKeyChecking")
	if err != nil {
		return result, err
	}

	connection, err := sshConnect(id.Host, port, user, strictHostKeyChecking != "no")
	if err != nil {
		return result, fmt.Errorf("Failed to connect: %s", err)
	}

	// Run command via session
	session, err := connection.NewSession()
	if err != nil {
		return result, fmt.Errorf("Failed to create session: %s", err)
	}

	if opts.Tty {
		// sshd only processes signal codes if a TTY has been allocated. So set one up when requested.
		modes := ssh.TerminalModes{ssh.ECHO: 0, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
		if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
			return result, fmt.Errorf("Unable to setup a TTY: %v", err)
		}
	}

	outputch, stdout, stderr := plugin.CreateExecOutputStreams(ctx)
	session.Stdin, session.Stdout, session.Stderr = opts.Stdin, stdout, stderr

	cmdStr := shellquote.Join(cmd...)
	if err := session.Start(cmdStr); err != nil {
		return result, err
	}

	// Wait for session to complete and stash result. This is done asynchronously because the
	// exitcode callback won't be called until stdout/stderr are closed, but they're not closed by
	// session.Start.
	var exitErr error
	go func() {
		exitErr = session.Wait()
		activity.Record(ctx, "Closing session for %v: %v", id.Host, session.Close())
		stdout.Close()
		stderr.Close()
	}()

	result.OutputCh = outputch
	result.CancelFunc = func() {
		// Close the session on context cancellation. Copying will block until there's more to read
		// from the exec output. For an action with no more output it may never return.
		// If a TTY is setup and the session is still open, send Ctrl-C over before closing it.
		if opts.Tty {
			activity.Record(ctx, "Sent SIGINT on context termination: %v", session.Signal(ssh.SIGINT))
		}
		activity.Record(ctx, "Closing session on context termination for %v: %v", id.Host, session.Close())
	}
	result.ExitCodeCB = func() (int, error) {
		if exitErr == nil {
			return 0, nil
		} else if err, ok := exitErr.(*ssh.ExitError); ok {
			return err.ExitStatus(), nil
		}
		return 0, exitErr
	}
	return result, nil
}
