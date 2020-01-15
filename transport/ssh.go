package transport

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/avast/retry-go"
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

type sshConfig struct {
	host, port, user string
	identityFiles    []string
	hostKeyCallback  ssh.HostKeyCallback
}

func getConnInfo(ctx context.Context, id Identity) (conf sshConfig, err error) {
	if conf.host, err = ssh_config.GetStrict(id.Host, "HostName"); err != nil {
		return
	}
	if conf.host == "" {
		conf.host = id.Host
	}

	// ssh_config provides a default of port 22.
	if conf.port, err = ssh_config.GetStrict(id.Host, "Port"); err != nil {
		return
	}

	conf.user = id.User
	if conf.user == "" {
		if conf.user, err = ssh_config.GetStrict(id.Host, "User"); err != nil {
			return
		}
	}
	if conf.user == "" {
		conf.user = id.FallbackUser
	}
	if conf.user == "" {
		conf.user = "root"
	}

	// Try the requested identity file first. Include any in SSH config as well just-in-case.
	conf.identityFiles = make([]string, 0)
	if id.IdentityFile != "" {
		conf.identityFiles = append(conf.identityFiles, id.IdentityFile)
	}
	if id.IdentityFile, err = ssh_config.GetStrict(id.Host, "IdentityFile"); err != nil {
		return
	}
	if id.IdentityFile != "" {
		conf.identityFiles = append(conf.identityFiles, id.IdentityFile)
	}
	// We later provide a fallback to ssh-agent if none of the provided identity files work.

	// Implement permissive and accept-new host key checking. Also account for HostKeyAlias.
	// Defaults to accepting new hosts.
	var hostKeyChecking string
	if hostKeyChecking, err = ssh_config.GetStrict(id.Host, "StrictHostKeyChecking"); err != nil {
		return
	}
	conf.hostKeyCallback = ssh.InsecureIgnoreHostKey()
	if hostKeyChecking == "no" {
		// Return early. This must be the last field configured.
		return
	}

	if id.KnownHosts == "" {
		var homedir string
		if homedir, err = os.UserHomeDir(); err != nil {
			return
		}

		id.KnownHosts = filepath.Join(homedir, ".ssh", "known_hosts")
	}
	// The known hosts file must exist before we try to use it.
	var f *os.File
	if f, err = os.OpenFile(id.KnownHosts, os.O_RDONLY|os.O_CREATE, 0644); err != nil {
		return
	}
	f.Close()

	conf.hostKeyCallback, err = knownhosts.New(id.KnownHosts)
	if err != nil {
		err = fmt.Errorf("Loading SSH known hosts file: %v", err)
		return
	}
	conf.hostKeyCallback = acceptNewCallback(ctx, conf.hostKeyCallback, id.KnownHosts)

	// Lookup host key alias for use in key checking. This should be the last wrapped so that
	// other callbacks use the alias.
	if id.HostKeyAlias == "" {
		if id.HostKeyAlias, err = ssh_config.GetStrict(id.Host, "HostKeyAlias"); err != nil {
			return
		}
	}
	if id.HostKeyAlias != "" {
		conf.hostKeyCallback = hostAliasCallback(conf.hostKeyCallback, id.HostKeyAlias)
	}
	return
}

func sshConnect(ctx context.Context, conf sshConfig, retries uint) (*ssh.Client, error) {
	connID := conf.user + "@" + conf.host + ":" + conf.port
	// This is a single-use cache, so pass in an empty category.
	obj, err := connectionCache.GetOrUpdate("", connID, expires, true, func() (interface{}, error) {
		agent, err := newAgent()
		if err != nil {
			return nil, fmt.Errorf("Failed to find config from SSH_AUTH_SOCK: %v", err)
		}

		var authmethod []ssh.AuthMethod
		for _, identityFile := range conf.identityFiles {
			if key, err := ioutil.ReadFile(identityFile); err != nil {
				activity.Record(ctx, "Unable to read private key, falling back to SSH agent: %v", err)
			} else {
				if signer, err := ssh.ParsePrivateKey(key); err != nil {
					activity.Record(ctx, "Unable to parse private key, falling back to SSH agent: %v", err)
				} else {
					authmethod = append(authmethod, ssh.PublicKeys(signer))
				}
			}
		}
		// Append agent now so that it comes last in case we find another method to try.
		authmethod = append(authmethod, agent)
		sshConfig := &ssh.ClientConfig{
			User:            conf.user,
			Auth:            authmethod,
			HostKeyCallback: conf.hostKeyCallback,
		}

		// Try until we've retried desired number of times or connection is established.
		var cli *ssh.Client
		err = retry.Do(
			func() error {
				cli, err = ssh.Dial("tcp", conf.host+":"+conf.port, sshConfig)
				return err
			},
			retry.Attempts(retries+1),
			retry.Delay(500*time.Millisecond),
		)
		return cli, err
	})

	if err != nil {
		return nil, err
	}
	return obj.(*ssh.Client), nil
}

// Identity identifies how to connect to a target.
type Identity struct {
	Host, User, FallbackUser, IdentityFile, KnownHosts, HostKeyAlias string
	// Retries can be set to a non-zero value to retry every 500ms for that many times.
	Retries uint
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
//   Host *.compute.amazonaws.com
//     StrictHostKeyChecking no
func ExecSSH(ctx context.Context, id Identity, cmd []string, opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	// find port, username, etc from .ssh/config
	conf, err := getConnInfo(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("Failed to get connection info: %s", err)
	}
	activity.Record(ctx, "Found connection info %+v", conf)

	connection, err := sshConnect(ctx, conf, id.Retries)
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

func hostAliasCallback(cb ssh.HostKeyCallback, hostKeyAlias string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		_, port, err := net.SplitHostPort(hostname)
		if err != nil {
			return err
		}

		return cb(net.JoinHostPort(hostKeyAlias, port), remote, key)
	}
}

func acceptNewCallback(ctx context.Context, cb ssh.HostKeyCallback, knownHosts string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := cb(hostname, remote, key)
		if err != nil {
			// If the error occurred because no entry was found, add it to known hosts and succeed.
			if kerr, ok := err.(*knownhosts.KeyError); ok && len(kerr.Want) == 0 {
				line := knownhosts.Line([]string{hostname}, key)
				if err := appendToKnownHosts(ctx, knownHosts, line); err != nil {
					activity.Warnf(ctx, "Unable to update %v with new host %v: %v", knownHosts, hostname, err)
				}
				return nil
			}
		}
		return err
	}
}

func appendToKnownHosts(ctx context.Context, knownHosts, line string) error {
	f, err := os.OpenFile(knownHosts, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		return err
	}
	return nil
}
