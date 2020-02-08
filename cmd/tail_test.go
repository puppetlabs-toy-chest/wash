package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLineWriter(t *testing.T) {
	// Setup a writer with test message and output channel. Also mark time before any messages.
	const msg = "a complete line"
	out := make(chan line, 1)
	lw := lineWriter{name: "mine", out: out}

	// Write a message, mark after it's written but before we finish to verify Write produced it.
	start := time.Now()
	validWrite(t, &lw, msg+"\n")
	mark := time.Now()
	lw.Finish()
	assertLine(t, out, "mine", msg, start, mark)

	start = time.Now()
	validWrite(t, &lw, msg+"\r")
	mark = time.Now()
	lw.Finish()
	assertLine(t, out, "mine", msg, start, mark)

	// Classic Windows endings, e.g. CRLF
	start = time.Now()
	validWrite(t, &lw, msg+"\r\n")
	mark = time.Now()
	lw.Finish()
	assertLine(t, out, "mine", msg, start, mark)

	// Test message split over multiple writes
	start = time.Now()
	split := len(msg) / 2
	validWrite(t, &lw, msg[:split])
	validWrite(t, &lw, msg[split:])
	validWrite(t, &lw, "\r\n")
	mark = time.Now()
	lw.Finish()
	assertLine(t, out, "mine", msg, start, mark)

	// Test multiple lines, with no newline on last one
	start = time.Now()
	validWrite(t, &lw, msg)
	validWrite(t, &lw, "\r")
	assertLine(t, out, "mine", msg, start, time.Now())
	start = time.Now()
	validWrite(t, &lw, msg)
	validWrite(t, &lw, "\n")
	assertLine(t, out, "mine", msg, start, time.Now())
	start = time.Now()
	validWrite(t, &lw, msg)
	lw.Finish()
	assertLine(t, out, "mine", msg, start, time.Now())
}

func validWrite(t *testing.T, lw *lineWriter, msg string) {
	n, err := lw.Write([]byte(msg))
	assert.NoError(t, err)
	assert.Equal(t, len(msg), n)
}

func assertLine(t *testing.T, out <-chan line, source, msg string, before, after time.Time) {
	ln, ok := <-out
	assert.True(t, ok)
	assert.NoError(t, ln.Err)
	assert.Equal(t, "mine", ln.source)
	assert.Equal(t, msg, ln.Text)
	assert.True(t, before.Before(ln.Time))
	assert.True(t, after.After(ln.Time))
}
