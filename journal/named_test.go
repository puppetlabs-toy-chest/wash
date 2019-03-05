package journal

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamedJournal(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer std.Flush()

	nj := NamedJournal{"0"}
	// Log to a journal
	nj.Log("hello there")

	bits, err := ioutil.ReadFile(filepath.Join(journaldir(), "0.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello there")
	}
}

func TestUnknownName(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer std.Flush()

	nj := NamedJournal{}
	// Log to a journal
	nj.Log("hello %v", "there")

	bits, err := ioutil.ReadFile(filepath.Join(journaldir(), "dead-letter-office.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "hello there")
	}
}
