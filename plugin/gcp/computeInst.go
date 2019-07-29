package gcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/puppetlabs/wash/activity"
	"github.com/puppetlabs/wash/plugin"
	"github.com/puppetlabs/wash/transport"
	"golang.org/x/crypto/ssh"
	compute "google.golang.org/api/compute/v1"
)

type computeInstance struct {
	plugin.EntryBase
	instance *compute.Instance
	service  computeProjectService
}

func newComputeInstance(inst *compute.Instance, c computeProjectService) *computeInstance {
	comp := &computeInstance{
		EntryBase: plugin.NewEntry(inst.Name),
		instance:  inst,
		service:   c,
	}
	comp.Attributes().SetMeta(inst)
	return comp
}

func (c *computeInstance) List(ctx context.Context) ([]plugin.Entry, error) {
	return []plugin.Entry{newComputeInstanceConsoleOutput(c.instance, c.service)}, nil
}

func (c *computeInstance) Schema() *plugin.EntrySchema {
	return plugin.NewEntrySchema(c, "instance").SetMetaAttributeSchema(compute.Instance{})
}

func (c *computeInstance) ChildSchemas() []*plugin.EntrySchema {
	return []*plugin.EntrySchema{
		(&computeInstanceConsoleOutput{}).Schema(),
	}
}

func (c *computeInstance) Exec(ctx context.Context, cmd string, args []string,
	opts plugin.ExecOptions) (plugin.ExecCommand, error) {
	conf, err := gceSSHFiles()
	if err != nil {
		return nil, err
	}

	// Extract username and key from public key file name.
	user, key, err := parseUserAndKey(conf.publicKey)
	if err != nil {
		return nil, err
	}

	keyAdded, err := c.addPublicKey(ctx, user, key)
	if err != nil {
		return nil, err
	}

	// TODO: Get host keys for the instance.
	// See https://github.com/google-cloud-sdk/google-cloud-sdk/blob/v255.0.0/lib/googlecloudsdk/command_lib/compute/ssh_utils.py#L501-L507
	// It's not clear when and how keys are added to guest attributes, so I haven't set this up.

	// Exec with associated private key.
	activity.Record(ctx, "Found user %v for %v", user, c.Name())

	hostname := getExternalIP(c.instance)
	if hostname == "" {
		return nil, fmt.Errorf("%v does not have an external IP address", c.Name())
	}

	identity := transport.Identity{
		Host:         hostname,
		FallbackUser: user,
		IdentityFile: conf.privateKey,
		KnownHosts:   conf.knownHosts,
		HostKeyAlias: hostKeyAlias(c.instance),
	}
	if keyAdded {
		// It may take some time for the new key to be added to the instance. Retry for up to 15s.
		identity.Retries = 30
	}
	return transport.ExecSSH(ctx, identity, append([]string{cmd}, args...), opts)
}

// Based on https://github.com/google-cloud-sdk/google-cloud-sdk/blob/v255.0.0/lib/googlecloudsdk/command_lib/compute/ssh_utils.py#L106
func getExternalIP(instance *compute.Instance) string {
	for _, intf := range instance.NetworkInterfaces {
		if len(intf.AccessConfigs) > 0 && intf.AccessConfigs[0].NatIP != "" {
			return intf.AccessConfigs[0].NatIP
		}
	}
	return ""
}

// Based on https://github.com/google-cloud-sdk/google-cloud-sdk/blob/v255.0.0/lib/googlecloudsdk/command_lib/compute/ssh_utils.py#L889
func hostKeyAlias(instance *compute.Instance) string {
	return fmt.Sprintf("compute.%v", instance.Id)
}

func parseUserAndKey(pubKeyFile string) (string, string, error) {
	content, err := ioutil.ReadFile(pubKeyFile)
	if err != nil {
		return "", "", fmt.Errorf("could not read SSH public key: %v", err)
	}

	key := strings.TrimSpace(string(content))
	userStart := strings.LastIndex(key, " ")
	userEnd := strings.LastIndex(key, "@")
	if userEnd < userStart {
		userEnd = len(key)
	}
	return key[userStart+1 : userEnd], key, nil
}

func getZone(instance *compute.Instance) string {
	// Zone is given as a URL on the Instance type.
	zoneSlice := strings.Split(instance.Zone, "/")
	return zoneSlice[len(zoneSlice)-1]
}

// The gcloud CLI tool works by convention, where it uses key and known hosts files at this
// location unless otherwise configured. We'll do the same: if they exist use them, if not create
// them and use them. Use $USER when generating a new key.
type gceSSH struct {
	privateKey, publicKey, knownHosts string
}

func gceSSHFiles() (gceSSH, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return gceSSH{}, fmt.Errorf("could not find SSH keys: %v", err)
	}

	keys := gceSSH{
		privateKey: filepath.Join(homeDir, ".ssh", "google_compute_engine"),
		publicKey:  filepath.Join(homeDir, ".ssh", "google_compute_engine.pub"),
		knownHosts: filepath.Join(homeDir, ".ssh", "google_compute_known_hosts"),
	}

	// Generate new keys if they don't exist.
	_, err1 := os.Stat(keys.publicKey)
	_, err2 := os.Stat(keys.privateKey)
	if err1 != nil || err2 != nil {
		if os.IsNotExist(err1) && os.IsNotExist(err2) {
			if err := generateKeys(keys.publicKey, keys.privateKey); err != nil {
				return keys, fmt.Errorf("failed to generate new SSH keys: %v", err)
			}
		} else {
			// This is a bad state, error instead of overwriting any keys.
			return keys, fmt.Errorf("errors reading %v - %v - and %v - %v",
				keys.publicKey, err1, keys.privateKey, err2)
		}
	}

	return keys, nil
}

func generateKeys(publicKeyPath, privateKeyPath string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	if err = privateKey.Validate(); err != nil {
		return err
	}

	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privKeyBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privDER,
	})

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicKey)
	hostname, err := os.Hostname()
	if err != nil {
		return err
	}
	user := " " + os.Getenv("USER") + "@" + hostname + "\n"
	pubKeyBytes = append(bytes.TrimSpace(pubKeyBytes), []byte(user)...)

	if err := ioutil.WriteFile(privateKeyPath, privKeyBytes, 0600); err != nil {
		return err
	}

	if err := ioutil.WriteFile(publicKeyPath, pubKeyBytes, 0600); err != nil {
		return err
	}

	return nil
}

// SSH public keys are added to Common Instance Metadata as described in
// https://cloud.google.com/compute/docs/instances/adding-removing-ssh-keys
//
// Adds the key to metadata in the prescribed format (mapping user to public key content) so that it
// will be added to `~/.ssh/authorized_keys` for the supplied user on associated instances.
// Returns true if the key was added, false if it was already present.
//
// The following is modeled on
// https://github.com/google-cloud-sdk/google-cloud-sdk/blob/v256.0.0/lib/googlecloudsdk/command_lib/compute/ssh_utils.py#L679-L706.
//
// The VM grabs keys from the metadata as follows (pseudo-code):
//     if 'sshKeys' in instance.metadata:
//       return instance.metadata['sshKeys'] + instance.metadata['ssh-keys']
//     elif instance.metadata['block-project-ssh-keys'] == 'true':
//       return instance.metadata['ssh-keys']
//     else:
//       return instance.metadata['ssh-keys'] + project.metadata['ssh-keys'] + project.metadata['sshKeys']
//
// Once a key exists (we may have created it earlier) we
// 1. If sshKeys exists in instance metadata or block-project-ssh-keys is true,
//    ensure the key is in instance.metadata['ssh-keys']
// 2. Else ensure the key is in project.metadata['ssh-keys']
// 3. If (2) failed (lacking permission), ensure the key is in instance.metadata['ssh-keys']
func (c *computeInstance) addPublicKey(ctx context.Context, user, key string) (bool, error) {
	uploadInstanceMetadata := func(metadata *compute.Metadata) error {
		_, err := c.service.Instances.SetMetadata(c.service.projectID, getZone(c.instance), c.Name(), metadata).Context(ctx).Do()
		return err
	}

	if legacySSHKeys := findKey(c.instance.Metadata, legacySSHKey); legacySSHKeys != nil {
		return ensureKey(c.instance.Metadata, newSSHKey, user, key, uploadInstanceMetadata)
	}

	if blockProjectSSH := findKey(c.instance.Metadata, blockProjectSSHKey); blockProjectSSH != nil && *blockProjectSSH == "true" {
		return ensureKey(c.instance.Metadata, newSSHKey, user, key, uploadInstanceMetadata)
	}

	// Try adding the key to project metadata.
	proj, err := c.service.Projects.Get(c.service.projectID).Context(ctx).Do()
	if err != nil {
		return false, err
	}

	uploadProjectMetadata := func(metadata *compute.Metadata) error {
		_, err := c.service.Projects.SetCommonInstanceMetadata(c.service.projectID, metadata).Context(ctx).Do()
		return err
	}

	keyAdded, err := ensureKey(proj.CommonInstanceMetadata, newSSHKey, user, key, uploadProjectMetadata)
	if err == nil {
		// The key is now present, so return.
		return keyAdded, err
	}

	// Unable to update project metadata.
	activity.Record(ctx, "Unable to add SSH key to metadata for project %v: %v", c.service.projectID, err)

	// Try adding the key to instance metadata.
	keyAdded, err = ensureKey(c.instance.Metadata, newSSHKey, user, key, uploadInstanceMetadata)
	if err == nil {
		// The key is now present, so return.
		return keyAdded, err
	}

	return false, fmt.Errorf("unable to add SSH key to instance metadata for %v: %v", c, err)
}

const (
	legacySSHKey       = "sshKeys"
	newSSHKey          = "ssh-keys"
	blockProjectSSHKey = "block-project-ssh-keys"
)

func findKey(metadata *compute.Metadata, key string) *string {
	var value *string
	for _, item := range metadata.Items {
		if item.Key == key {
			value = item.Value
		}
	}
	return value
}

// Returns true if the key was added, false if it was already present.
func ensureKey(metadata *compute.Metadata, keyField, user, key string, upload func(*compute.Metadata) error) (bool, error) {
	newKeyLine := user + ":" + key

	sshKeys := findKey(metadata, keyField)
	if sshKeys == nil {
		// No prior metadata item was found, add a new one.
		sshKeysEntry := compute.MetadataItems{Key: keyField, Value: &newKeyLine}
		metadata.Items = append(metadata.Items, &sshKeysEntry)
	} else if strings.Contains(*sshKeys, newKeyLine) {
		// Key is already present, skip adding it.
		return false, nil
	} else {
		// Found a metadata item, append to it.
		if *sshKeys != "" && (*sshKeys)[len(*sshKeys)-1] != '\n' {
			*sshKeys += "\n"
		}
		*sshKeys += newKeyLine
	}
	err := upload(metadata)
	return err == nil, err
}
