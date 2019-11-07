package activity

import (
	"context"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Reads one line at a time.
type lineReader struct {
	string
}

func (r *lineReader) Read(p []byte) (int, error) {
	src := r.string
	if src == "" {
		return 0, io.EOF
	}

	i := strings.Index(src, "\n")
	if i == -1 {
		r.string = ""
	} else if i < len(p) {
		// Add 1 to include the newline.
		r.string = src[i+1:]
		src = src[:i+1]
	} else {
		r.string = src[len(p):]
	}

	n := copy(p, src)
	return n, nil
}

func TestWriter(t *testing.T) {
	// Ensure the cache is cleaned up afterward.
	defer CloseAll()

	// Log to a journal
	ctx := context.WithValue(context.Background(), JournalKey, Journal{ID: "testWriter"})
	wr := Writer{Context: ctx, Prefix: "line"}

	const message = "some text\nmore text\n\nand even more\n"
	rdr := lineReader{message}
	n, err := io.Copy(wr, &rdr)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(message)), n)

	bits, err := ioutil.ReadFile(filepath.Join(Dir(), "testWriter.log"))
	if assert.Nil(t, err) {
		assert.Contains(t, string(bits), "line: some text")
		assert.Contains(t, string(bits), "line: more text")
		assert.Contains(t, string(bits), "line: ")
		assert.Contains(t, string(bits), "line: and even more")
	}
}
