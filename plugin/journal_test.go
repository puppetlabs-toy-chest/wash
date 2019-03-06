package plugin

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/puppetlabs/wash/journal"
	"github.com/stretchr/testify/assert"
)

func TestRecord(t *testing.T) {
	msg := "hello"
	Record(context.WithValue(context.Background(), Journal, "42"), msg)

	bits, err := ioutil.ReadFile(filepath.Join(journal.Dir(), "42.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), msg)
	}
}

func TestDeadLetterOffice(t *testing.T) {
	Record(context.WithValue(context.Background(), Journal, ""), "hello %v", "world")

	bits, err := ioutil.ReadFile(filepath.Join(journal.Dir(), "dead-letter-office.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello world")
	}
}

func TestLogging(t *testing.T) {
	Record(context.Background(), "nobody home")

	bits, err := ioutil.ReadFile(filepath.Join(journal.Dir(), "dead-letter-office.log"))
	// Could get an error if dead-letter-office does not exist.
	if err == nil {
		assert.NotContains(t, string(bits), "nobody home")
	}
}

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "plugin_tests")
	if err != nil {
		panic(err)
	}
	journal.SetDir(dir)

	exitcode := m.Run()

	os.RemoveAll(dir)
	os.Exit(exitcode)
}
