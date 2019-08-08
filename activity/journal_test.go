package activity

import (
	"context"
	"io/ioutil"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHistory(t *testing.T) {
	// Ensure history is empty
	history = initHistory()

	// Clean up tests at the end.
	defer func() {
		history = initHistory()
		CloseAll()
	}()

	assert.Equal(t, []Journal{}, History())

	tick := time.Now()
	journal := Journal{ID: "anything", Description: "did something", start: tick}
	journal.addToHistory()

	assert.Equal(t, []Journal{journal}, History())

	journal.addToHistory()
	assert.Equal(t, []Journal{journal}, History())

	_, err := journal.Open()
	assert.Error(t, err)
}

func TestHistoryWithJournal(t *testing.T) {
	// Ensure history is empty
	history = initHistory()

	// Clean up tests at the end.
	defer func() {
		history = initHistory()
		CloseAll()
	}()

	// Log to a journal
	journal := Journal{ID: "anything"}
	ctx := context.WithValue(context.Background(), JournalKey, journal)
	Record(ctx, "hello there")
	Warnf(ctx, "not good")
	rdr, err := journal.Open()
	if assert.Nil(t, err) {
		defer rdr.Close()
		bits, err := ioutil.ReadAll(rdr)
		if assert.Nil(t, err) {
			assert.Regexp(t, regexp.MustCompile("info.*hello there"), string(bits))
			assert.Regexp(t, regexp.MustCompile("warn.*not good"), string(bits))
		}
	}
}

func TestRecorder_CanRecordMethodInvocations(t *testing.T) {
	recorder := newRecorder()

	assert.False(t, recorder.methodInvoked("foo", "List"))
	recorder.recordMethodInvocation("foo", "List")
	assert.True(t, recorder.methodInvoked("foo", "List"))

	// Test a different method
	assert.False(t, recorder.methodInvoked("foo", "Exec"))
	recorder.recordMethodInvocation("foo", "Exec")
	assert.True(t, recorder.methodInvoked("foo", "Exec"))
}
