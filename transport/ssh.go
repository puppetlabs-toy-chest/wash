package transport

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"io/ioutil"
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
var connectionCache = datastore.NewMemCache().WithEvicted(closeConnection)
var expires = 15 * time.Second

func closeConnection(id string, obj interface{}) {
	if client, ok := obj.(*ssh.Client); ok {
		client.Close()
	}
}

func newAgent() (ssh.AuthMethod, error) {
	sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))

	if err != nil {
		return nil, err
	}
	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers), nil
}

func getHostKeyCallback() (ssh.HostKeyCallback, error) {
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return knownhosts.New(filepath.Join(homedir, ".ssh", "known_hosts"))
}

func sshConnect(ctx context.Context, host, port, user string, identityfile string, strictHostKeyChecking bool) (*ssh.Client, error) {
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

		var authmethod []ssh.AuthMethod
		if key, err := ioutil.ReadFile(identityfile); err != nil {
			activity.Record(ctx, "Unable to read private key, falling back to SSH agent: %v", err)
		} else {
			if signer, err := ssh.ParsePrivateKey(key); err != nil {
				activity.Record(ctx, "Unable to parse private key, falling back to SSH agent: %v", err)
			} else {
				authmethod = append(authmethod, ssh.PublicKeys(signer))
			}
		}
		// Append agent now so that it comes last in case we find another method to try.
		authmethod = append(authmethod, agent)
		sshConfig := &ssh.ClientConfig{
			User:            user,
			Auth:            authmethod,
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
	Host, User, FallbackUser, IdentityFile string
}

// ExecSSH executes against a target via SSH. It will look up port, user, and other configuration
// by exact hostname match from default SSH config files. Identity can be used to override the
// user configured in SSH config. If opts.Elevate is true, will attempt to `sudo` as root.
//
// If present, a local SSH agent will be used for authentication.
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
func ExecSSH(ctx context.Context, id Identity, cmd []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	// find port, username, etc from .ssh/config
	port, err := ssh_config.GetStrict(id.Host, "Port")
	if err != nil {
		return nil, err
	}

	user := id.User
	identityfile := id.IdentityFile
	if user == "" {
		if user, err = ssh_config.GetStrict(id.Host, "User"); err != nil {
			return nil, err
		}
	}

	if user == "" {
		user = id.FallbackUser
	}

	if user == "" {
		user = "root"
	}
	strictHostKeyChecking, err := ssh_config.GetStrict(id.Host, "StrictHostKeyChecking")
	if err != nil {
		return nil, err
	}

	connection, err := sshConnect(ctx, id.Host, port, user, identityfile, strictHostKeyChecking != "no")
	if err != nil {
		return nil, fmt.Errorf("Failed to connect: %s", err)
	}

	// Run command via session
	session, err := connection.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}

	if opts.Tty {
		// sshd only processes signal codes if a TTY has been allocated. So set one up when requested.
		modes := ssh.TerminalModes{ssh.ECHO: 0, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
		if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
			return nil, fmt.Errorf("Unable to setup a TTY: %v", err)
		}
	}

	execCmd := plugin.NewExecCommand(ctx)
	session.Stdin, session.Stdout, session.Stderr = opts.Stdin, execCmd.Stdout(), execCmd.Stderr()

	if opts.Elevate {
		cmd = append([]string{"sudo"}, cmd...)
	}

	cmdStr := shellquote.Join(cmd...)
	if err := session.Start(cmdStr); err != nil {
		return nil, err
	}
	execCmd.SetStopFunc(func() {
		// Close the session on context cancellation. Copying will block until there's more to read
		// from the exec output. For an action with no more output it may never return.
		// If a TTY is setup and the session is still open, send Ctrl-C over before closing it.
		if opts.Tty {
			activity.Record(ctx, "Sent SIGINT on context termination: %v", session.Signal(ssh.SIGINT))
		}
		activity.Record(ctx, "Closing session on context termination for %v: %v", id.Host, session.Close())
	})

	// Wait for session to complete and stash result.
	go func() {
		err := session.Wait()
		activity.Record(ctx, "Closing session for %v: %v", id.Host, session.Close())
		execCmd.CloseStreamsWithError(nil)
		if err == nil {
			execCmd.SetExitCode(0)
		} else if exitErr, ok := err.(*ssh.ExitError); ok {
			execCmd.SetExitCode(exitErr.ExitStatus())
		} else {
			execCmd.SetExitCodeErr(err)
		}
	}()
	return execCmd, nil
}
