package journal

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRecord(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer journalCache.Flush()

	// Log to a journal
	Record(context.WithValue(context.Background(), Key, "1"), "hello there")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "1.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello there")
	}
}

func TestLogExpired(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer journalCache.Flush()

	// Ensure entries use a very short
	expires = 1 * time.Millisecond
	ctx := context.WithValue(context.Background(), Key, "2")

	// Log twice, second after cache entry has expired
	Record(ctx, "first write")
	time.Sleep(1 * time.Millisecond)
	Record(ctx, "second write")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "2.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestLogReused(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer journalCache.Flush()
	ctx := context.WithValue(context.Background(), Key, "3")

	// Log twice
	Record(ctx, "first write")
	Record(ctx, "second %v", "write")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "3.log"))
	if assert.Nil(t, err) {
		assert.Regexp(t, "(?s)first write.*second write", string(bits))
	}
}

func TestDeadLetterOffice(t *testing.T) {
	Record(context.WithValue(context.Background(), Key, ""), "hello %v", "world")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "dead-letter-office.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello world")
	}
}

func TestLogging(t *testing.T) {
	Record(context.Background(), "nobody home")

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "dead-letter-office.log"))
	// Could get an error if dead-letter-office does not exist.
	if err == nil {
		assert.NotContains(t, string(bits), "nobody home")
	}
}

func TestMain(m *testing.M) {
	dir, err := ioutil.TempDir("", "journal_tests")
	if err != nil {
		panic(err)
	}
	SetDir(dir)

	exitcode := m.Run()

	os.RemoveAll(dir)
	os.Exit(exitcode)
}
