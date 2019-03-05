package journal

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLog(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer std.Flush()

	// Log to a journal
	Log("42", "hello there")

	bits, err := ioutil.ReadFile(filepath.Join(journaldir(), "42.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello there")
	}
}

func TestLogExpired(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer std.Flush()

	// Ensure entries use a very short
	expires = 1 * time.Millisecond

	// Log twice, second after cache entry has expired
	Log("1", "first write")
	time.Sleep(1 * time.Millisecond)
	Log("1", "second write")

	bits, err := ioutil.ReadFile(filepath.Join(journaldir(), "1.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestLogReused(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer std.Flush()

	// Log twice
	Log("2", "first write")
	Log("2", "second %v", "write")

	bits, err := ioutil.ReadFile(filepath.Join(journaldir(), "2.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestMain(m *testing.M) {
	var err error
	if cachedir, err = ioutil.TempDir("", "journal_tests"); err != nil {
		panic(err)
	}

	exitcode := m.Run()

	os.RemoveAll(cachedir)
	os.Exit(exitcode)
}
