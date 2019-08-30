package gcp

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/puppetlabs/wash/plugin"
	"github.com/stretchr/testify/assert"
	compute "google.golang.org/api/compute/v1"
)

func TestComputeInstance(t *testing.T) {
	inst := compute.Instance{Name: "foo", CreationTimestamp: time.Now().Format(time.RFC3339) }
	compInst := newComputeInstance(&inst, computeProjectService{})
	assert.Equal(t, "foo", compInst.Name())
	assert.Implements(t, (*plugin.Parent)(nil), compInst)
	assert.Implements(t, (*plugin.Execable)(nil), compInst)
}

func TestParseUserAndKey(t *testing.T) {
	// Exercise generateKeys to create temporary test keys.
	keyDir, err := ioutil.TempDir("", "computeInstTest_ParseUserAndKey")
	assert.NoError(t, err)
	defer os.RemoveAll(keyDir)

	pubKeyPath := filepath.Join(keyDir, "mykey.pub")
	privKeyPath := filepath.Join(keyDir, "mykey")
	assert.NoError(t, generateKeys(pubKeyPath, privKeyPath))

	user, key, err := parseUserAndKey(pubKeyPath)
	assert.NoError(t, err)
	assert.Equal(t, os.Getenv("USER"), user)

	elements := strings.Split(strings.TrimSpace(key), " ")
	assert.Equal(t, 3, len(elements))
	assert.Equal(t, "ssh-rsa", elements[0])
	assert.NotEmpty(t, elements[1])

	hostname, err := os.Hostname()
	assert.NoError(t, err)
	userString := os.Getenv("USER") + "@" + hostname
	assert.Equal(t, userString, elements[2])

	// Remove the address from the written public key file and test that it still works.
	content, err := ioutil.ReadFile(pubKeyPath)
	assert.NoError(t, err)
	userEnd := bytes.LastIndex(content, []byte{'@'})
	assert.NoError(t, ioutil.WriteFile(pubKeyPath, content[0:userEnd], 0600))

	user, key, err = parseUserAndKey(pubKeyPath)
	assert.NoError(t, err)
	assert.Equal(t, os.Getenv("USER"), user)

	elements = strings.Split(strings.TrimSpace(key), " ")
	assert.Equal(t, 3, len(elements))
	assert.Equal(t, "ssh-rsa", elements[0])
	assert.NotEmpty(t, elements[1])
	assert.Equal(t, os.Getenv("USER"), elements[2])
}

func TestGetZone(t *testing.T) {
	inst := compute.Instance{Zone: "https://some/path/to/zone/myzone"}
	assert.Equal(t, "myzone", getZone(&inst))
}

func TestFindKey(t *testing.T) {
	meta := compute.Metadata{}
	assert.Nil(t, findKey(&meta, "my-key"))

	testString := "some value"
	meta.Items = append(meta.Items, &compute.MetadataItems{Key: "my-key", Value: &testString})
	assert.Equal(t, &testString, findKey(&meta, "my-key"))
	assert.Nil(t, findKey(&meta, "not-my-key"))
}
